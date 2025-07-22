package geoip

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"net"

	"github.com/oschwald/geoip2-golang"
)

// Embedded GeoLite2-City database (zipped for smaller binary size)
// Original size: ~60MB, Compressed: ~30MB, Binary size reduction: ~47%
//
//go:embed GeoLite2-City.mmdb.zip
var embeddedDB []byte

// Database wraps the GeoIP2 database
type Database struct {
	db *geoip2.Reader
}

// Location represents a geographic location
type Location struct {
	Latitude  float64
	Longitude float64
	Country   string
	City      string
}

// NewDatabase creates a new GeoIP database instance
// If dbPath is empty, uses embedded database; otherwise uses external file
func NewDatabase(dbPath string) (*Database, error) {
	var db *geoip2.Reader
	var err error

	if dbPath == "" {
		// Use embedded database (zipped)
		db, err = loadEmbeddedDatabase()
		if err != nil {
			return nil, fmt.Errorf("failed to open embedded GeoIP database: %w", err)
		}
	} else {
		// Use external database file
		db, err = geoip2.Open(dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open GeoIP database from %s: %w", dbPath, err)
		}
	}

	return &Database{db: db}, nil
}

// NewEmbeddedDatabase creates a new GeoIP database instance using embedded data
func NewEmbeddedDatabase() (*Database, error) {
	return NewDatabase("")
}

// loadEmbeddedDatabase decompresses and loads the embedded zipped database
func loadEmbeddedDatabase() (*geoip2.Reader, error) {
	// Create a reader from the embedded zip data
	zipReader, err := zip.NewReader(bytes.NewReader(embeddedDB), int64(len(embeddedDB)))
	if err != nil {
		return nil, fmt.Errorf("failed to create zip reader: %w", err)
	}

	// Find the .mmdb file in the zip
	var mmdbFile *zip.File
	for _, file := range zipReader.File {
		if file.Name == "GeoLite2-City.mmdb" ||
			(len(file.Name) > 5 && file.Name[len(file.Name)-5:] == ".mmdb") {
			mmdbFile = file
			break
		}
	}

	if mmdbFile == nil {
		return nil, fmt.Errorf("no .mmdb file found in embedded zip")
	}

	// Open and read the mmdb file from the zip
	rc, err := mmdbFile.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open mmdb file from zip: %w", err)
	}

	// Read all data into memory
	mmdbData, err := io.ReadAll(rc)
	closeErr := rc.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read mmdb data: %w", err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("failed to close mmdb file: %w", closeErr)
	}

	// Create the geoip2 reader from the decompressed data
	db, err := geoip2.FromBytes(mmdbData)
	if err != nil {
		return nil, fmt.Errorf("failed to create geoip2 reader: %w", err)
	}

	return db, nil
}

// Close closes the database
func (d *Database) Close() error {
	return d.db.Close()
}

// Lookup looks up the location for an IP address
func (d *Database) Lookup(ipStr string) (*Location, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	record, err := d.db.City(ip)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup IP %s: %w", ipStr, err)
	}

	location := &Location{
		Latitude:  record.Location.Latitude,
		Longitude: record.Location.Longitude,
	}

	if len(record.Country.Names) > 0 {
		location.Country = record.Country.Names["en"]
	}

	if len(record.City.Names) > 0 {
		location.City = record.City.Names["en"]
	}

	return location, nil
}
