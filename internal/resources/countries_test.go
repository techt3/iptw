package resources

import (
	"sort"
	"strings"
	"testing"
)

func TestGetAlpha2ByName(t *testing.T) {
	tests := []struct {
		name        string
		countryName string
		expected    string
		shouldError bool
	}{
		{
			name:        "United States",
			countryName: "United States of America",
			expected:    "US",
			shouldError: false,
		},
		{
			name:        "Afghanistan",
			countryName: "Afghanistan",
			expected:    "AF",
			shouldError: false,
		},
		{
			name:        "Case insensitive",
			countryName: "ALBANIA",
			expected:    "AL",
			shouldError: false,
		},
		{
			name:        "Non-existent country",
			countryName: "Atlantis",
			expected:    "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetAlpha2ByName(tt.countryName)

			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetNameByAlpha2(t *testing.T) {
	tests := []struct {
		name        string
		alpha2      string
		expected    string
		shouldError bool
	}{
		{
			name:        "US code",
			alpha2:      "US",
			expected:    "United States of America",
			shouldError: false,
		},
		{
			name:        "AF code",
			alpha2:      "AF",
			expected:    "Afghanistan",
			shouldError: false,
		},
		{
			name:        "Case insensitive",
			alpha2:      "al",
			expected:    "Albania",
			shouldError: false,
		},
		{
			name:        "Invalid code",
			alpha2:      "XX",
			expected:    "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetNameByAlpha2(tt.alpha2)

			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetCountryByAlpha2(t *testing.T) {
	country, err := GetCountryByAlpha2("US")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if country.Name != "United States of America" {
		t.Errorf("expected 'United States of America', got %s", country.Name)
	}

	if country.Alpha3 != "USA" {
		t.Errorf("expected 'USA', got %s", country.Alpha3)
	}
}

func TestGetAllCountries(t *testing.T) {
	countries := GetAllCountries()
	if len(countries) == 0 {
		t.Error("expected countries to be loaded")
	}

	// Should have around 249 countries based on the CSV
	if len(countries) < 240 {
		t.Errorf("expected at least 240 countries, got %d", len(countries))
	}

	// Check that Afghanistan is first (based on CSV structure)
	if len(countries) > 0 && countries[0].Name != "Afghanistan" {
		t.Errorf("expected first country to be Afghanistan, got %s", countries[0].Name)
	}
}

// TestCountryNameMatching compares country names from Natural Earth JSON and countries CSV
func TestCountryNameMatching(t *testing.T) {
	// Load Natural Earth data
	neData, err := LoadNaturalEarthData()
	if err != nil {
		t.Fatalf("failed to load Natural Earth data: %v", err)
	}

	// Get all countries from CSV
	csvCountries := GetAllCountries()
	if len(csvCountries) == 0 {
		t.Fatal("no countries loaded from CSV")
	}

	// Create maps for easier lookup
	neCountryNames := make(map[string]bool)
	csvCountryNames := make(map[string]bool)

	// Normalize and collect Natural Earth country names
	for _, country := range neData.Countries {
		normalizedName := normalizeCountryName(country.Name)
		if normalizedName != "" {
			neCountryNames[normalizedName] = true
		}
	}

	// Normalize and collect CSV country names
	for _, country := range csvCountries {
		normalizedName := normalizeCountryName(country.Name)
		if normalizedName != "" {
			csvCountryNames[normalizedName] = true
		}
	}

	// Find matching records
	var matchingRecords []string
	var neOnlyRecords []string
	var csvOnlyRecords []string

	// Check for matches and Natural Earth only records
	for neName := range neCountryNames {
		if csvCountryNames[neName] {
			matchingRecords = append(matchingRecords, neName)
		} else {
			neOnlyRecords = append(neOnlyRecords, neName)
		}
	}

	// Check for CSV only records
	for csvName := range csvCountryNames {
		if !neCountryNames[csvName] {
			csvOnlyRecords = append(csvOnlyRecords, csvName)
		}
	}

	// Sort all slices for consistent output
	sort.Strings(matchingRecords)
	sort.Strings(neOnlyRecords)
	sort.Strings(csvOnlyRecords)

	// Log results
	t.Logf("Total Natural Earth countries: %d", len(neCountryNames))
	t.Logf("Total CSV countries: %d", len(csvCountryNames))
	t.Logf("Matching records: %d", len(matchingRecords))
	t.Logf("Natural Earth only: %d", len(neOnlyRecords))
	t.Logf("CSV only: %d", len(csvOnlyRecords))

	// Print matching records
	if len(matchingRecords) > 0 {
		t.Logf("\nMatching country names (%d):", len(matchingRecords))
		for i, name := range matchingRecords {
			if i < 20 { // Limit output to first 20 for readability
				t.Logf("  - %s", name)
			} else if i == 20 {
				t.Logf("  ... and %d more", len(matchingRecords)-20)
				break
			}
		}
	}

	// Print Natural Earth only records
	if len(neOnlyRecords) > 0 {
		t.Logf("\nCountries in Natural Earth but not in CSV (%d):", len(neOnlyRecords))
		for i, name := range neOnlyRecords {
			if i < 20 { // Limit output to first 20 for readability
				t.Logf("  - %s", name)
			} else if i == 20 {
				t.Logf("  ... and %d more", len(neOnlyRecords)-20)
				break
			}
		}
	}

	// Print CSV only records
	if len(csvOnlyRecords) > 0 {
		t.Logf("\nCountries in CSV but not in Natural Earth (%d):", len(csvOnlyRecords))
		for i, name := range csvOnlyRecords {
			if i < 20 { // Limit output to first 20 for readability
				t.Logf("  - %s", name)
			} else if i == 20 {
				t.Logf("  ... and %d more", len(csvOnlyRecords)-20)
				break
			}
		}
	}

	// Assertions - we expect some matches but also some differences
	if len(matchingRecords) == 0 {
		t.Error("expected at least some matching country names between Natural Earth and CSV")
	}

	// Log coverage percentage
	matchPercentage := float64(len(matchingRecords)) / float64(len(neCountryNames)) * 100
	t.Logf("\nMatch coverage: %.1f%% of Natural Earth countries have matches in CSV", matchPercentage)

	// Store results in test context for potential debugging
	if t.Failed() {
		t.Logf("Test failed - see logs above for detailed comparison results")
	}
}

// normalizeCountryName normalizes country names for comparison
func normalizeCountryName(name string) string {
	// Convert to lowercase and trim spaces
	normalized := strings.ToLower(strings.TrimSpace(name))

	// Handle common variations
	replacements := map[string]string{
		"united states of america":                             "united states",
		"united kingdom of great britain and northern ireland": "united kingdom",
		"russian federation":                                   "russia",
		"iran, islamic republic of":                            "iran",
		"korea, republic of":                                   "south korea",
		"korea, democratic people's republic of":               "north korea",
		"venezuela, bolivarian republic of":                    "venezuela",
		"bolivia, plurinational state of":                      "bolivia",
		"tanzania, united republic of":                         "tanzania",
		"moldova, republic of":                                 "moldova",
		"micronesia, federated states of":                      "micronesia",
		"congo, democratic republic of the":                    "democratic republic of the congo",
		"palestine, state of":                                  "palestine",
		"taiwan, province of china":                            "taiwan",
		"netherlands, kingdom of the":                          "netherlands",
		"bonaire, sint eustatius and saba":                     "bonaire",
		"saint helena, ascension and tristan da cunha":         "saint helena",
		"virgin islands (british)":                             "british virgin islands",
		"virgin islands (u.s.)":                                "us virgin islands",
		"cocos (keeling) islands":                              "cocos islands",
		"heard island and mcdonald islands":                    "heard island",
		"south georgia and the south sandwich islands":         "south georgia",
		"svalbard and jan mayen":                               "svalbard",
		"turks and caicos islands":                             "turks and caicos",
		"wallis and futuna":                                    "wallis and futuna islands",
	}

	// Apply replacements
	if replacement, exists := replacements[normalized]; exists {
		normalized = replacement
	}

	// Remove common prefixes/suffixes that might cause mismatches
	normalized = strings.TrimPrefix(normalized, "the ")
	normalized = strings.TrimSuffix(normalized, " islands")
	normalized = strings.TrimSuffix(normalized, " island")

	return normalized
}

// TestDetailedCountryComparison provides detailed analysis of country name differences
func TestDetailedCountryComparison(t *testing.T) {
	// Load Natural Earth data
	neData, err := LoadNaturalEarthData()
	if err != nil {
		t.Fatalf("failed to load Natural Earth data: %v", err)
	}

	// Get all countries from CSV
	csvCountries := GetAllCountries()
	if len(csvCountries) == 0 {
		t.Fatal("no countries loaded from CSV")
	}

	// Create detailed comparison structures
	type CountryComparison struct {
		NaturalEarthName string
		CSVName          string
		Alpha2Code       string
		MatchType        string
	}

	var comparisons []CountryComparison

	// Create maps for analysis
	neCountryNames := make(map[string]string)   // normalized -> original
	csvCountriesMap := make(map[string]Country) // normalized -> Country

	// Process Natural Earth data
	for _, country := range neData.Countries {
		normalized := normalizeCountryName(country.Name)
		if normalized != "" {
			neCountryNames[normalized] = country.Name
		}
	}

	// Process CSV data
	for _, country := range csvCountries {
		normalized := normalizeCountryName(country.Name)
		if normalized != "" {
			csvCountriesMap[normalized] = country
		}
	}

	// Find exact matches
	for normalizedName, originalNEName := range neCountryNames {
		if csvCountry, exists := csvCountriesMap[normalizedName]; exists {
			comparisons = append(comparisons, CountryComparison{
				NaturalEarthName: originalNEName,
				CSVName:          csvCountry.Name,
				Alpha2Code:       csvCountry.Alpha2,
				MatchType:        "EXACT_MATCH",
			})
		}
	}

	// Find Natural Earth only entries
	for normalizedName, originalNEName := range neCountryNames {
		if _, exists := csvCountriesMap[normalizedName]; !exists {
			comparisons = append(comparisons, CountryComparison{
				NaturalEarthName: originalNEName,
				CSVName:          "",
				Alpha2Code:       "",
				MatchType:        "NE_ONLY",
			})
		}
	}

	// Find CSV only entries
	for normalizedName, csvCountry := range csvCountriesMap {
		if _, exists := neCountryNames[normalizedName]; !exists {
			comparisons = append(comparisons, CountryComparison{
				NaturalEarthName: "",
				CSVName:          csvCountry.Name,
				Alpha2Code:       csvCountry.Alpha2,
				MatchType:        "CSV_ONLY",
			})
		}
	}

	// Sort comparisons by match type and name
	sort.Slice(comparisons, func(i, j int) bool {
		if comparisons[i].MatchType != comparisons[j].MatchType {
			return comparisons[i].MatchType < comparisons[j].MatchType
		}
		// Sort by whichever name is available
		nameI := comparisons[i].NaturalEarthName
		if nameI == "" {
			nameI = comparisons[i].CSVName
		}
		nameJ := comparisons[j].NaturalEarthName
		if nameJ == "" {
			nameJ = comparisons[j].CSVName
		}
		return nameI < nameJ
	})

	// Count by type
	var exactMatches, neOnly, csvOnly int
	for _, comp := range comparisons {
		switch comp.MatchType {
		case "EXACT_MATCH":
			exactMatches++
		case "NE_ONLY":
			neOnly++
		case "CSV_ONLY":
			csvOnly++
		}
	}

	// Log summary
	t.Logf("=== DETAILED COUNTRY COMPARISON RESULTS ===")
	t.Logf("Total comparisons: %d", len(comparisons))
	t.Logf("Exact matches: %d", exactMatches)
	t.Logf("Natural Earth only: %d", neOnly)
	t.Logf("CSV only: %d", csvOnly)
	t.Logf("")

	// Show some examples of each type
	currentType := ""
	shown := 0
	maxPerType := 10

	for _, comp := range comparisons {
		if comp.MatchType != currentType {
			currentType = comp.MatchType
			shown = 0
			t.Logf("=== %s ===", currentType)
		}

		if shown < maxPerType {
			switch comp.MatchType {
			case "EXACT_MATCH":
				if comp.NaturalEarthName == comp.CSVName {
					t.Logf("  %s (%s) - identical names", comp.NaturalEarthName, comp.Alpha2Code)
				} else {
					t.Logf("  NE: '%s' <-> CSV: '%s' (%s)", comp.NaturalEarthName, comp.CSVName, comp.Alpha2Code)
				}
			case "NE_ONLY":
				t.Logf("  '%s' - exists in Natural Earth but not in CSV", comp.NaturalEarthName)
			case "CSV_ONLY":
				t.Logf("  '%s' (%s) - exists in CSV but not in Natural Earth", comp.CSVName, comp.Alpha2Code)
			}
			shown++
		} else if shown == maxPerType {
			remaining := 0
			for _, c := range comparisons {
				if c.MatchType == currentType {
					remaining++
				}
			}
			remaining -= maxPerType
			if remaining > 0 {
				t.Logf("  ... and %d more %s entries", remaining, strings.ToLower(strings.ReplaceAll(currentType, "_", " ")))
			}
			shown++
		}
	}

	// Calculate coverage statistics
	coveragePercent := float64(exactMatches) / float64(len(neCountryNames)) * 100
	t.Logf("")
	t.Logf("=== COVERAGE STATISTICS ===")
	t.Logf("Natural Earth coverage: %.1f%% (%d/%d countries have matches in CSV)",
		coveragePercent, exactMatches, len(neCountryNames))

	csvCoveragePercent := float64(exactMatches) / float64(len(csvCountriesMap)) * 100
	t.Logf("CSV coverage: %.1f%% (%d/%d countries have matches in Natural Earth)",
		csvCoveragePercent, exactMatches, len(csvCountriesMap))

	// Test assertions
	if exactMatches == 0 {
		t.Error("expected at least some exact matches between datasets")
	}

	if exactMatches < len(neCountryNames)/2 {
		t.Errorf("low match rate: only %d/%d Natural Earth countries matched", exactMatches, len(neCountryNames))
	}
}

// TestAnalyzeMismatchedEntries analyzes all mismatched entries for potential fixes
func TestAnalyzeMismatchedEntries(t *testing.T) {
	// Load Natural Earth data
	neData, err := LoadNaturalEarthData()
	if err != nil {
		t.Fatalf("failed to load Natural Earth data: %v", err)
	}

	// Get all countries from CSV
	csvCountries := GetAllCountries()
	if len(csvCountries) == 0 {
		t.Fatal("no countries loaded from CSV")
	}

	// Create maps for analysis
	neCountryNames := make(map[string]string)   // normalized -> original
	csvCountriesMap := make(map[string]Country) // normalized -> Country
	csvByAlpha2 := make(map[string]Country)     // alpha2 -> Country
	csvByName := make(map[string]Country)       // original name -> Country

	// Process Natural Earth data
	for _, country := range neData.Countries {
		normalized := normalizeCountryName(country.Name)
		if normalized != "" {
			neCountryNames[normalized] = country.Name
		}
	}

	// Process CSV data
	for _, country := range csvCountries {
		normalized := normalizeCountryName(country.Name)
		if normalized != "" {
			csvCountriesMap[normalized] = country
		}
		csvByAlpha2[country.Alpha2] = country
		csvByName[country.Name] = country
	}

	t.Logf("=== ANALYZING MISMATCHED ENTRIES FOR POTENTIAL FIXES ===")
	t.Logf("")

	// Analyze Natural Earth only entries
	t.Logf("=== NATURAL EARTH ONLY ENTRIES (need CSV equivalents) ===")
	neOnlyCount := 0
	for normalizedName, originalNEName := range neCountryNames {
		if _, exists := csvCountriesMap[normalizedName]; !exists {
			neOnlyCount++
			t.Logf("%d. NE: '%s'", neOnlyCount, originalNEName)

			// Try to find potential matches in CSV by similarity
			possibleMatches := findSimilarCSVEntries(originalNEName, csvCountries)
			if len(possibleMatches) > 0 {
				t.Logf("   Possible CSV matches:")
				for i, match := range possibleMatches {
					if i < 3 { // Show max 3 suggestions
						t.Logf("     - '%s' (%s)", match.Name, match.Alpha2)
					}
				}
			}
			t.Logf("")
		}
	}

	t.Logf("=== SUMMARY OF COMMON MISMATCHES ===")

	// Analyze common patterns
	commonMismatches := map[string]string{
		"Brunei":                      "Brunei Darussalam",
		"Czech Republic":              "Czechia",
		"East Timor":                  "Timor-Leste",
		"Falkland Islands":            "Falkland Islands (Malvinas)",
		"Guinea Bissau":               "Guinea-Bissau",
		"Ivory Coast":                 "Côte d'Ivoire",
		"Laos":                        "Lao People's Democratic Republic",
		"Macedonia":                   "North Macedonia",
		"Syria":                       "Syrian Arab Republic",
		"Turkey":                      "Türkiye",
		"USA":                         "United States of America",
		"Republic of the Congo":       "Congo",
		"United Republic of Tanzania": "Tanzania, United Republic of",
		"Swaziland":                   "Eswatini",
	}

	t.Logf("Common name variations that could be standardized:")
	for neVariant, csvName := range commonMismatches {
		if csvCountry, exists := csvByName[csvName]; exists {
			t.Logf("  NE: '%s' -> CSV: '%s' (%s)", neVariant, csvName, csvCountry.Alpha2)
		}
	}
}

// findSimilarCSVEntries finds CSV entries that might match a Natural Earth country name
func findSimilarCSVEntries(neName string, csvCountries []Country) []Country {
	var matches []Country
	neLower := strings.ToLower(neName)

	for _, csvCountry := range csvCountries {
		csvLower := strings.ToLower(csvCountry.Name)

		// Check for partial matches or contains
		if strings.Contains(csvLower, neLower) || strings.Contains(neLower, csvLower) {
			matches = append(matches, csvCountry)
			continue
		}

		// Check for word-based similarity
		neWords := strings.Fields(neLower)
		csvWords := strings.Fields(csvLower)

		commonWords := 0
		for _, neWord := range neWords {
			for _, csvWord := range csvWords {
				if neWord == csvWord ||
					(len(neWord) > 3 && len(csvWord) > 3 &&
						(strings.Contains(neWord, csvWord) || strings.Contains(csvWord, neWord))) {
					commonWords++
					break
				}
			}
		}

		// If more than half the words match, consider it a potential match
		if commonWords > 0 && float64(commonWords)/float64(len(neWords)) > 0.5 {
			matches = append(matches, csvCountry)
		}
	}

	return matches
}
