// Package ispdetect discovers all IP address ranges that belong to the user's
// Internet Service Provider.  Detection works in three steps:
//
//  1. Resolve the machine's public (WAN) IP address using a lightweight HTTP
//     probe against well-known "what-is-my-ip" endpoints.
//  2. Identify the ISP's Autonomous System Number (ASN) for that IP via the
//     RIPE Stat network-info API (covers all registries globally).
//  3. Fetch every BGP prefix announced by that ASN via RIPE Stat
//     announced-prefixes API, so ALL of the ISP's address space is covered.
//
// If the ASN-based path fails, a fallback RDAP lookup returns at least the
// single network block that contains the public IP.
//
// The collected CIDRs are used to filter out connections to/from the user's
// own ISP infrastructure so they do not appear on the travel map.
package ispdetect

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

var publicIPProviders = []string{
	"https://api.ipify.org",
	"https://icanhazip.com",
	"https://ipecho.net/plain",
}

// GetPublicIP returns the machine's public WAN IP address.
// It tries each provider in publicIPProviders and returns the first success.
func GetPublicIP() (string, error) {
	for _, url := range publicIPProviders {
		ip, err := fetchPublicIP(url)
		if err == nil {
			return ip, nil
		}
		slog.Debug("ispdetect: public-IP provider failed", "url", url, "error", err)
	}
	return "", fmt.Errorf("all public-IP providers failed")
}

func fetchPublicIP(url string) (string, error) {
	resp, err := httpClient.Get(url) //nolint:gosec // URL is from a hardcoded allowlist
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(body))
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("response is not a valid IP: %q", ip)
	}
	return ip, nil
}

// rdapIPResponse is a minimal representation of the RDAP ip-network object.
// https://tools.ietf.org/html/rfc7483#section-5.4
type rdapIPResponse struct {
	StartAddress string `json:"startAddress"`
	EndAddress   string `json:"endAddress"`
	CIDRs        []struct {
		V4Prefix string `json:"v4prefix"`
		V6Prefix string `json:"v6prefix"`
		Length   int    `json:"length"`
	} `json:"cidr0_cidrs"`
}

// ripeStatAnnouncedPrefixes is a partial representation of the RIPE Stat
// announced-prefixes response:
// https://stat.ripe.net/docs/02.data-api/announced-prefixes.html
type ripeStatAnnouncedPrefixes struct {
	Data struct {
		Prefixes []struct {
			Prefix string `json:"prefix"`
		} `json:"prefixes"`
	} `json:"data"`
}

const (
	rdapBootstrapURL        = "https://rdap.org/ip/"
	ripeStatAnnouncedPfxURL = "https://stat.ripe.net/data/announced-prefixes/data.json?resource="
)

// getASNForIP resolves the originating ASN for the given IP address using the
// Team Cymru IP-to-ASN DNS mapping service (https://team-cymru.com/community-services/ip-asn-mapping/).
//
// The IP is embedded in a DNS hostname (e.g. 155.151.209.37.origin.asn.cymru.com)
// and resolved via the system's normal DNS resolver.  No HTTP request containing
// the IP address is sent to any third-party service.
//
// Returns the ASN in "AS12345" format.
func getASNForIP(ip string) (string, error) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "", fmt.Errorf("invalid IP address: %s", ip)
	}

	var dnsName string
	if v4 := parsed.To4(); v4 != nil {
		// Reverse the four octets: 37.209.151.155 → 155.151.209.37.origin.asn.cymru.com
		dnsName = fmt.Sprintf("%d.%d.%d.%d.origin.asn.cymru.com", v4[3], v4[2], v4[1], v4[0])
	} else {
		// IPv6: expand to 32 hex nibbles, reverse, join with dots.
		full := parsed.To16()
		nibbles := make([]string, 32)
		for i, b := range full {
			nibbles[31-(i*2)] = fmt.Sprintf("%x", b&0x0f)
			nibbles[31-(i*2+1)] = fmt.Sprintf("%x", b>>4)
		}
		dnsName = strings.Join(nibbles, ".") + ".origin6.asn.cymru.com"
	}

	txts, err := net.LookupTXT(dnsName)
	if err != nil {
		return "", fmt.Errorf("Cymru DNS lookup for %s failed: %w", dnsName, err)
	}
	if len(txts) == 0 {
		return "", fmt.Errorf("no TXT records for %s", dnsName)
	}

	// TXT format: "29167 | 37.209.144.0/20 | PL | ripencc | 2006-11-07"
	raw := strings.TrimSpace(txts[0])
	fields := strings.SplitN(raw, "|", 2)
	asn := strings.TrimSpace(fields[0])
	if asn == "" {
		return "", fmt.Errorf("empty ASN in TXT record: %q", raw)
	}
	return "AS" + asn, nil
}

// getASNPrefixes fetches all BGP prefixes announced by the given ASN from
// RIPE Stat.  The ASN must be in "AS12345" form.
func getASNPrefixes(asn string) ([]*net.IPNet, error) {
	url := ripeStatAnnouncedPfxURL + asn
	resp, err := httpClient.Get(url) //nolint:gosec // constructed from a validated ASN string
	if err != nil {
		return nil, fmt.Errorf("RIPE Stat announced-prefixes request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RIPE Stat announced-prefixes returned HTTP %d for %s", resp.StatusCode, asn)
	}
	var data ripeStatAnnouncedPrefixes
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&data); err != nil {
		return nil, fmt.Errorf("RIPE Stat announced-prefixes JSON decode failed: %w", err)
	}

	var networks []*net.IPNet
	for _, p := range data.Data.Prefixes {
		_, network, err := net.ParseCIDR(p.Prefix)
		if err != nil {
			slog.Debug("ispdetect: skipping unparseable prefix", "prefix", p.Prefix, "error", err)
			continue
		}
		networks = append(networks, network)
	}
	if len(networks) == 0 {
		return nil, fmt.Errorf("RIPE Stat returned no prefixes for %s", asn)
	}
	return networks, nil
}

// DetectISPCIDRs returns all BGP-announced prefixes belonging to the ISP that
// owns publicIP.  It uses RIPE Stat to get the ASN and all its prefixes.
// If the ASN-based path fails, it falls back to a single RDAP CIDR lookup.
func DetectISPCIDRs(publicIP string) ([]*net.IPNet, error) {
	// Primary path: ASN → all announced prefixes.
	asn, err := getASNForIP(publicIP)
	if err != nil {
		slog.Warn("ispdetect: ASN lookup failed, falling back to RDAP single-CIDR", "error", err)
	} else {
		slog.Info("ispdetect: detected ISP ASN", "asn", asn)
		networks, err := getASNPrefixes(asn)
		if err != nil {
			slog.Warn("ispdetect: ASN prefix lookup failed, falling back to RDAP single-CIDR", "error", err)
		} else {
			return networks, nil
		}
	}

	// Fallback path: RDAP single network block for publicIP.
	return detectISPCIDRsViaRDAP(publicIP)
}

// detectISPCIDRsViaRDAP is the original RDAP-based fallback that returns only
// the single network block registered for the given IP.
func detectISPCIDRsViaRDAP(publicIP string) ([]*net.IPNet, error) {
	url := rdapBootstrapURL + publicIP
	resp, err := httpClient.Get(url) //nolint:gosec // constructed from a validated IP string
	if err != nil {
		return nil, fmt.Errorf("RDAP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RDAP returned HTTP %d for %s", resp.StatusCode, publicIP)
	}
	var data rdapIPResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&data); err != nil {
		return nil, fmt.Errorf("RDAP JSON decode failed: %w", err)
	}

	var networks []*net.IPNet
	for _, c := range data.CIDRs {
		var cidr string
		if c.V4Prefix != "" {
			cidr = fmt.Sprintf("%s/%d", c.V4Prefix, c.Length)
		} else if c.V6Prefix != "" {
			cidr = fmt.Sprintf("%s/%d", c.V6Prefix, c.Length)
		}
		if cidr == "" {
			continue
		}
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			slog.Debug("ispdetect: skipping unparseable CIDR", "cidr", cidr, "error", err)
			continue
		}
		networks = append(networks, network)
	}

	if len(networks) == 0 && data.StartAddress != "" && data.EndAddress != "" {
		networks = append(networks, rangeToNetworks(data.StartAddress, data.EndAddress)...)
	}

	if len(networks) == 0 {
		return nil, fmt.Errorf("RDAP response contained no usable network prefixes for %s", publicIP)
	}
	return networks, nil
}

// DetectISPCIDRsAuto detects the public IP then calls DetectISPCIDRs.
// It logs the results at INFO level and is the primary entry point for callers.
func DetectISPCIDRsAuto() ([]*net.IPNet, error) {
	publicIP, err := GetPublicIP()
	if err != nil {
		return nil, fmt.Errorf("could not determine public IP: %w", err)
	}
	slog.Info("ispdetect: detected public IP", "ip", publicIP)

	cidrs, err := DetectISPCIDRs(publicIP)
	if err != nil {
		return nil, fmt.Errorf("could not detect ISP CIDRs for %s: %w", publicIP, err)
	}
	slog.Info("ispdetect: ISP filtering active", "cidr_count", len(cidrs))
	for _, c := range cidrs {
		slog.Debug("ispdetect: ISP CIDR registered", "cidr", c.String())
	}
	return cidrs, nil
}

// rangeToNetworks converts an IP range (startAddress-endAddress) to the
// minimal list of covering CIDR blocks.  Used as a fallback when the RDAP
// response lacks the cidr0 extension.
func rangeToNetworks(start, end string) []*net.IPNet {
	startIP := net.ParseIP(start)
	endIP := net.ParseIP(end)
	if startIP == nil || endIP == nil {
		return nil
	}
	if v4 := startIP.To4(); v4 != nil {
		startIP = v4
	}
	if v4 := endIP.To4(); v4 != nil {
		endIP = v4
	}

	var networks []*net.IPNet
	cur := cloneIP(startIP)
	for ipLE(cur, endIP) {
		bits := len(cur) * 8
		prefixLen := bits
		added := false
		for prefixLen > 0 {
			prefixLen--
			_, network, err := net.ParseCIDR(fmt.Sprintf("%s/%d", cur.String(), prefixLen))
			if err != nil {
				prefixLen++
				break
			}
			last := lastIP(network)
			if ipLE(last, endIP) {
				networks = append(networks, network)
				cur = nextIP(last)
				added = true
				break
			}
		}
		if !added {
			mask := net.CIDRMask(bits, bits)
			networks = append(networks, &net.IPNet{IP: cloneIP(cur), Mask: mask})
			cur = nextIP(cur)
		}
		if len(networks) > 256 {
			// Safety valve: avoid runaway loops on malformed RDAP data.
			break
		}
	}
	return networks
}

func cloneIP(ip net.IP) net.IP {
	c := make(net.IP, len(ip))
	copy(c, ip)
	return c
}

func ipLE(a, b net.IP) bool {
	for i := range a {
		if a[i] < b[i] {
			return true
		}
		if a[i] > b[i] {
			return false
		}
	}
	return true // equal
}

func lastIP(network *net.IPNet) net.IP {
	ip := cloneIP(network.IP)
	for i := range ip {
		ip[i] |= ^network.Mask[i]
	}
	return ip
}

func nextIP(ip net.IP) net.IP {
	next := cloneIP(ip)
	for i := len(next) - 1; i >= 0; i-- {
		next[i]++
		if next[i] != 0 {
			break
		}
	}
	return next
}
