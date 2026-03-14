// Package factdb provides a "Did you know?" fact database of obscure,
// genuinely interesting facts about countries and cities.
//
// Facts are stored at two levels:
//   - Country-level: unusual records, historical quirks, counterintuitive
//     geography, cultural inversions, little-known firsts.
//   - City-level: facts specific to a city or sub-national location.
//
// The seed database is embedded in the binary (seed_facts.json), so facts
// are always available offline with zero network activity. City-level facts
// are tried first; if none exist, a country-level fact is returned.
package factdb

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
)

//go:embed seed_facts.json
var seedJSON []byte

// seedData is the raw JSON shape of seed_facts.json.
type seedData struct {
	Countries map[string][]string `json:"countries"`
	Cities    map[string][]string `json:"cities"`
	Regions   map[string][]string `json:"regions"`
}

// Fact is a single "Did you know?" entry.
type Fact struct {
	// Text is the fact sentence.
	Text string `json:"text"`
	// Level is "country", "city", or "region".
	Level string `json:"level"`
	// Place is the name of the country/city/region this fact is about.
	Place string `json:"place"`
}

// IsZero returns true when the Fact has no content.
func (f Fact) IsZero() bool { return f.Text == "" }

// DB is a fact database that returns random obscure facts for countries
// and cities. All data is loaded from the embedded seed JSON at construction
// time and is safe for concurrent use.
type DB struct {
	mu   sync.RWMutex
	data seedData
}

// New creates a DB loaded from the embedded seed_facts.json.
func New() (*DB, error) {
	var data seedData
	if err := json.Unmarshal(seedJSON, &data); err != nil {
		return nil, fmt.Errorf("factdb: failed to parse seed data: %w", err)
	}
	if data.Countries == nil {
		data.Countries = make(map[string][]string)
	}
	if data.Cities == nil {
		data.Cities = make(map[string][]string)
	}
	if data.Regions == nil {
		data.Regions = make(map[string][]string)
	}
	return &DB{
		data: data,
	}, nil
}

// GetFact returns a random interesting fact for the given country and,
// optionally, city. The lookup order is: city → country.
// Returns a zero Fact{} if no entry is found.
func (db *DB) GetFact(country, city string) Fact {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if city != "" {
		if f, ok := db.pick(db.data.Cities, city); ok {
			return Fact{Text: f, Level: "city", Place: city}
		}
	}
	if country != "" {
		if f, ok := db.pick(db.data.Countries, country); ok {
			return Fact{Text: f, Level: "country", Place: country}
		}
	}
	return Fact{}
}

// GetCountryFact is a convenience wrapper that ignores city.
func (db *DB) GetCountryFact(country string) Fact {
	return db.GetFact(country, "")
}

// CountryCount returns the number of countries with at least one fact.
func (db *DB) CountryCount() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.data.Countries)
}

// pick selects a uniformly random entry from m[key], returning ("", false)
// if the key does not exist or its slice is empty. The global rand source is
// used because it is concurrency-safe (locked internally since Go 1).
func (db *DB) pick(m map[string][]string, key string) (string, bool) {
	facts, ok := m[key]
	if !ok || len(facts) == 0 {
		return "", false
	}
	return facts[rand.Intn(len(facts))], true //nolint:gosec
}
