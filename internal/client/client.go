// Package client provides HTTP client functionality for fetching game statistics from iptw server
package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"iptw/internal/stats"
)

// Client represents the HTTP client for fetching game statistics
type Client struct {
	serverURL string
	timeout   time.Duration
}

// NewClient creates a new client instance
func NewClient(serverURL string) *Client {
	if serverURL == "" {
		serverURL = "http://localhost:32782" // Default server URL
	}
	return &Client{
		serverURL: serverURL,
		timeout:   10 * time.Second,
	}
}

// GetStats fetches game statistics from the server
func (c *Client) GetStats() (*stats.GameStatistics, error) {
	url := c.serverURL + "/stats/json"

	client := &http.Client{Timeout: c.timeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stats from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	gameStats, err := stats.FromJSON(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return gameStats, nil
}

// GetStatsText fetches game statistics as text from the server
func (c *Client) GetStatsText() (string, error) {
	url := c.serverURL + "/stats"

	client := &http.Client{Timeout: c.timeout}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch stats from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned status %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// GetAchievements fetches achievement details from the server
func (c *Client) GetAchievements() (string, error) {
	url := c.serverURL + "/achievements"

	client := &http.Client{Timeout: c.timeout}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch achievements from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned status %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// GetCountries fetches country visit details from the server
func (c *Client) GetCountries() (string, error) {
	url := c.serverURL + "/countries"

	client := &http.Client{Timeout: c.timeout}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch countries from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned status %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// CheckHealth performs a health check on the server
func (c *Client) CheckHealth() error {
	url := c.serverURL + "/health"

	client := &http.Client{Timeout: c.timeout}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to server at %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return fmt.Errorf("failed to parse health response: %w", err)
	}

	if status, ok := health["status"]; !ok || status != "healthy" {
		return fmt.Errorf("server reported unhealthy status: %v", health)
	}

	return nil
}

// PrintStats prints formatted game statistics
func (c *Client) PrintStats() error {
	gameStats, err := c.GetStats()
	if err != nil {
		return err
	}

	fmt.Println("IPTW Game Statistics")
	fmt.Println("===================")
	fmt.Println()
	fmt.Print(gameStats.Summary())
	fmt.Printf("\nServer: %s\n", c.serverURL)
	fmt.Printf("Updated: %s\n", gameStats.Timestamp.Format("2006-01-02 15:04:05"))

	return nil
}

// PrintAchievements prints formatted achievement information
func (c *Client) PrintAchievements() error {
	achievements, err := c.GetAchievements()
	if err != nil {
		return err
	}

	fmt.Print(achievements)
	fmt.Printf("\nServer: %s\n", c.serverURL)

	return nil
}

// PrintCountries prints formatted country visit information
func (c *Client) PrintCountries() error {
	countries, err := c.GetCountries()
	if err != nil {
		return err
	}

	fmt.Print(countries)
	fmt.Printf("\nServer: %s\n", c.serverURL)

	return nil
}

// WatchStats continuously polls and displays stats updates
func (c *Client) WatchStats(interval time.Duration) error {
	fmt.Printf("Watching IPTW stats from %s (polling every %v)\n", c.serverURL, interval)
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Print initial stats
	if err := c.PrintStats(); err != nil {
		return err
	}

	for range ticker.C {
		fmt.Println("\n" + strings.Repeat("=", 50))
		if err := c.PrintStats(); err != nil {
			fmt.Printf("Error fetching stats: %v\n", err)
			continue
		}
	}

	return nil
}

// Shutdown sends a shutdown request to the server
func (c *Client) Shutdown() error {
	url := c.serverURL + "/shutdown"

	client := &http.Client{Timeout: c.timeout}
	resp, err := client.Post(url, "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to send shutdown request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("shutdown request failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to parse shutdown response: %w", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		if errorMsg, exists := response["error"]; exists {
			return fmt.Errorf("shutdown failed: %v", errorMsg)
		}
		return fmt.Errorf("shutdown failed: %v", response)
	}

	return nil
}
