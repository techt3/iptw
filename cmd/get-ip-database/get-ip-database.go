package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// NOTE: As of the latest version, the GeoLite2-City database is embedded in the application.
// This tool is kept for reference and for updating the embedded database if needed.

const (
	downloadURL = "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=%s&suffix=tar.gz"
	tempDir     = "/tmp/geolite"
	tarFileName = "geolite.tar.gz"
	dbFileName  = "GeoLite2-City.mmdb"
)

func main() {
	if err := downloadGeoLiteDatabase(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func downloadGeoLiteDatabase() error {
	// Check if MaxMind license key is provided
	licenseKey := os.Getenv("MAXMIND_LICENSE_KEY")
	if licenseKey == "" {
		return fmt.Errorf("MAXMIND_LICENSE_KEY environment variable not set\n" +
			"Please obtain a free license key from https://www.maxmind.com/en/geolite2/signup\n" +
			"Then set it with: export MAXMIND_LICENSE_KEY=your_key_here")
	}

	// Create directories
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "iptw", "resources")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	tarFile := filepath.Join(tempDir, tarFileName)

	// Download the database if it doesn't exist
	if _, err := os.Stat(tarFile); os.IsNotExist(err) {
		fmt.Println("Downloading GeoLite2-City database...")
		if err := downloadFile(fmt.Sprintf(downloadURL, licenseKey), tarFile); err != nil {
			return fmt.Errorf("download failed: %w", err)
		}
		fmt.Println("Download completed successfully")
	} else {
		fmt.Println("Using existing downloaded database")
	}

	// Extract the database
	fmt.Println("Extracting database...")
	if err := extractDatabase(tarFile, tempDir, configDir); err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	fmt.Println("Setup completed successfully!")
	fmt.Println("You can now run: go run cmd/iptw/main.go")

	return nil
}

func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractDatabase(tarFile, tempDir, configDir string) error {
	file, err := os.Open(tarFile)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	var extractedDir string
	var mmdbFound bool

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(tempDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			// Track the main extracted directory
			if strings.Contains(header.Name, "GeoLite2-City_") && extractedDir == "" {
				extractedDir = target
			}

		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()

			// Check if this is the MMDB file we need
			if strings.HasSuffix(header.Name, dbFileName) {
				targetFile := filepath.Join(configDir, dbFileName)
				if err := copyFile(target, targetFile); err != nil {
					return err
				}
				fmt.Printf("Database installed to %s\n", targetFile)
				mmdbFound = true
			}
		}
	}

	if !mmdbFound {
		return fmt.Errorf("GeoLite2-City.mmdb not found in extracted archive")
	}

	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}
