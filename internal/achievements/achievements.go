// Package achievements provides achievement tracking for IP Travel Wallpaper
package achievements

import (
	"log/slog"
	"strings"
)

// Achievement represents a single achievement
type Achievement struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Unlocked    bool     `json:"unlocked"`
	Progress    int      `json:"progress"`
	Target      int      `json:"target"`
	Countries   []string `json:"countries,omitempty"`
}

// AchievementManager manages all achievements
type AchievementManager struct {
	achievements map[string]*Achievement
}

// NewAchievementManager creates a new achievement manager
func NewAchievementManager() *AchievementManager {
	am := &AchievementManager{
		achievements: make(map[string]*Achievement),
	}
	am.initializeAchievements()
	return am
}

// initializeAchievements sets up all available achievements
func (am *AchievementManager) initializeAchievements() {
	// Geographic Region Achievements
	am.achievements["europe_explorer"] = &Achievement{
		ID:          "europe_explorer",
		Name:        "European Explorer",
		Description: "Visit all countries in Europe",
		Target:      50, // Approximate number of European countries
		Countries:   getEuropeanCountries(),
	}

	am.achievements["asia_adventurer"] = &Achievement{
		ID:          "asia_adventurer",
		Name:        "Asian Adventurer",
		Description: "Visit all countries in Asia",
		Target:      50, // Approximate number of Asian countries
		Countries:   getAsianCountries(),
	}

	am.achievements["africa_explorer"] = &Achievement{
		ID:          "africa_explorer",
		Name:        "African Explorer",
		Description: "Visit all countries in Africa",
		Target:      54, // Number of African countries
		Countries:   getAfricanCountries(),
	}

	am.achievements["americas_wanderer"] = &Achievement{
		ID:          "americas_wanderer",
		Name:        "Americas Wanderer",
		Description: "Visit all countries in North and South America",
		Target:      35, // Approximate number of countries in the Americas
		Countries:   getAmericasCountries(),
	}

	am.achievements["oceania_voyager"] = &Achievement{
		ID:          "oceania_voyager",
		Name:        "Oceania Voyager",
		Description: "Visit all countries in Oceania",
		Target:      14, // Number of Oceanian countries
		Countries:   getOceaniaCountries(),
	}

	// Continental Achievements
	am.achievements["north_america_complete"] = &Achievement{
		ID:          "north_america_complete",
		Name:        "North American Complete",
		Description: "Visit all countries in North America",
		Target:      23,
		Countries:   getNorthAmericaCountries(),
	}

	am.achievements["south_america_complete"] = &Achievement{
		ID:          "south_america_complete",
		Name:        "South American Complete",
		Description: "Visit all countries in South America",
		Target:      12,
		Countries:   getSouthAmericaCountries(),
	}

	// Special Achievements
	am.achievements["world_traveler"] = &Achievement{
		ID:          "world_traveler",
		Name:        "World Traveler",
		Description: "Visit 100 different countries",
		Target:      100,
	}

	am.achievements["global_nomad"] = &Achievement{
		ID:          "global_nomad",
		Name:        "Global Nomad",
		Description: "Visit every country in the world",
		Target:      195, // Approximate number of UN recognized countries
	}

	am.achievements["rare_finder"] = &Achievement{
		ID:          "rare_finder",
		Name:        "Rare Destination Finder",
		Description: "Visit 10 rare or remote countries",
		Target:      10,
		Countries:   getRareCountries(),
	}
}

// UpdateProgress updates achievement progress when a country is visited
func (am *AchievementManager) UpdateProgress(countryName string, totalCountriesVisited int) []string {
	var newUnlocks []string

	for _, achievement := range am.achievements {
		if achievement.Unlocked {
			continue
		}

		// Update progress based on achievement type
		switch achievement.ID {
		case "world_traveler", "global_nomad":
			achievement.Progress = totalCountriesVisited
		default:
			// Region/continent specific achievements
			if achievement.Countries != nil {
				if containsCountry(achievement.Countries, countryName) {
					achievement.Progress++
				}
			}
		}

		// Check if achievement is now complete
		if achievement.Progress >= achievement.Target && !achievement.Unlocked {
			achievement.Unlocked = true
			newUnlocks = append(newUnlocks, achievement.ID)
			slog.Info("Achievement unlocked!",
				"achievement", achievement.Name,
				"description", achievement.Description,
			)
		}
	}

	return newUnlocks
}

// GetAllAchievements returns all achievements
func (am *AchievementManager) GetAllAchievements() map[string]*Achievement {
	return am.achievements
}

// GetUnlockedAchievements returns only unlocked achievements
func (am *AchievementManager) GetUnlockedAchievements() []*Achievement {
	var unlocked []*Achievement
	for _, achievement := range am.achievements {
		if achievement.Unlocked {
			unlocked = append(unlocked, achievement)
		}
	}
	return unlocked
}

// containsCountry checks if a country is in the list (case-insensitive)
func containsCountry(countries []string, country string) bool {
	country = strings.ToLower(country)
	for _, c := range countries {
		if strings.ToLower(c) == country {
			return true
		}
	}
	return false
}

// Geographic region definitions (simplified lists)
func getEuropeanCountries() []string {
	return []string{
		"Germany", "France", "Italy", "Spain", "United Kingdom", "Poland", "Romania",
		"Netherlands", "Belgium", "Czech Republic", "Greece", "Portugal", "Sweden",
		"Hungary", "Austria", "Belarus", "Switzerland", "Bulgaria", "Serbia", "Denmark",
		"Finland", "Slovakia", "Norway", "Ireland", "Croatia", "Bosnia and Herzegovina",
		"Albania", "Lithuania", "Slovenia", "Latvia", "Estonia", "Moldova", "Macedonia",
		"Luxembourg", "Malta", "Iceland", "Montenegro", "Cyprus", "Andorra", "Liechtenstein",
		"San Marino", "Monaco", "Vatican City", "Ukraine", "Russia",
	}
}

func getAsianCountries() []string {
	return []string{
		"China", "India", "Indonesia", "Pakistan", "Bangladesh", "Japan", "Philippines",
		"Vietnam", "Turkey", "Iran", "Thailand", "Myanmar", "South Korea", "Iraq",
		"Afghanistan", "Saudi Arabia", "Uzbekistan", "Malaysia", "Nepal", "Yemen",
		"North Korea", "Sri Lanka", "Kazakhstan", "Syria", "Cambodia", "Jordan",
		"Azerbaijan", "United Arab Emirates", "Tajikistan", "Israel", "Laos", "Singapore",
		"Oman", "Kuwait", "Georgia", "Mongolia", "Armenia", "Qatar", "Bahrain", "East Timor",
		"Palestine", "Lebanon", "Kyrgyzstan", "Bhutan", "Brunei", "Maldives",
	}
}

func getAfricanCountries() []string {
	return []string{
		"Nigeria", "Ethiopia", "Egypt", "Democratic Republic of the Congo", "Tanzania",
		"South Africa", "Kenya", "Uganda", "Algeria", "Sudan", "Morocco", "Angola",
		"Ghana", "Mozambique", "Madagascar", "Cameroon", "Côte d'Ivoire", "Niger",
		"Burkina Faso", "Mali", "Malawi", "Zambia", "Senegal", "Somalia", "Chad",
		"Zimbabwe", "Guinea", "Rwanda", "Benin", "Burundi", "Tunisia", "South Sudan",
		"Togo", "Sierra Leone", "Libya", "Liberia", "Central African Republic",
		"Mauritania", "Eritrea", "Gambia", "Botswana", "Namibia", "Gabon",
		"Lesotho", "Guinea-Bissau", "Equatorial Guinea", "Mauritius", "Eswatini",
		"Djibouti", "Comoros", "Cape Verde", "São Tomé and Príncipe", "Seychelles",
	}
}

func getAmericasCountries() []string {
	americas := append(getNorthAmericaCountries(), getSouthAmericaCountries()...)
	return americas
}

func getNorthAmericaCountries() []string {
	return []string{
		"United States", "Canada", "Mexico", "Guatemala", "Cuba", "Haiti",
		"Dominican Republic", "Honduras", "Nicaragua", "Costa Rica", "Panama",
		"Jamaica", "Trinidad and Tobago", "Belize", "Bahamas", "Barbados",
		"Saint Lucia", "Grenada", "Saint Vincent and the Grenadines",
		"Antigua and Barbuda", "Dominica", "Saint Kitts and Nevis", "El Salvador",
	}
}

func getSouthAmericaCountries() []string {
	return []string{
		"Brazil", "Argentina", "Colombia", "Peru", "Venezuela", "Chile",
		"Ecuador", "Bolivia", "Paraguay", "Uruguay", "Guyana", "Suriname",
	}
}

func getOceaniaCountries() []string {
	return []string{
		"Australia", "Papua New Guinea", "New Zealand", "Fiji", "Solomon Islands",
		"Vanuatu", "Samoa", "Kiribati", "Tonga", "Micronesia", "Palau",
		"Marshall Islands", "Tuvalu", "Nauru",
	}
}

func getRareCountries() []string {
	return []string{
		"Bhutan", "Mongolia", "Brunei", "San Marino", "Liechtenstein", "Monaco",
		"Vatican City", "Nauru", "Tuvalu", "Palau", "Marshall Islands",
		"Kiribati", "Andorra", "Luxembourg", "Malta",
	}
}
