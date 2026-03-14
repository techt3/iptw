// Package resources provides embedded static resources for the IP Travel Wallpaper application
package resources

import (
	"archive/zip"
	"bytes"
	"embed"
	"encoding/csv"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"math"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
)

//go:embed *.json *.zip *.csv Matrix-Code.ttf
var files embed.FS

// oceanCacheEntry holds a pre-rendered ocean background pixel buffer.
type oceanCacheEntry struct {
	pix    []uint8
	width  int
	height int
	dark   bool
}

var (
	oceanCacheMu sync.Mutex
	oceanCached  *oceanCacheEntry
)

// spanRun represents a horizontal span of pixels belonging to a country's rasterized shape.
type spanRun struct{ y, x1, x2 int }

// countryMaskEntry caches the rasterized pixel spans for one country at a specific resolution.
type countryMaskEntry struct {
	spans         []spanRun
	width, height int
}

var (
	countryMaskCacheMu sync.Mutex
	countryMaskCache   map[string]*countryMaskEntry
)

// CountryData represents a country with its geometry and metadata
type CountryData struct {
	Name     string
	Geometry orb.MultiPolygon
}

// Country represents country information from the CSV
type Country struct {
	Name                   string
	Alpha2                 string
	Alpha3                 string
	CountryCode            string
	ISO31662               string
	Region                 string
	SubRegion              string
	IntermediateRegion     string
	RegionCode             string
	SubRegionCode          string
	IntermediateRegionCode string
}

// CountryLookup manages country data and provides lookup functionality
type CountryLookup struct {
	countries    []Country
	nameToAlpha2 map[string]string
	alpha2ToName map[string]string
}

// NaturalEarthData holds all country data
type NaturalEarthData struct {
	Countries []CountryData
}

func (c *CountryData) getAlpha2Code() string {
	alpha2, err := GetAlpha2ByName(c.Name)
	if err != nil {
		slog.Warn("failed to load country data for", "name", c.Name, "error", err)
		return ""
	}
	return alpha2
}

// FontManager manages loaded fonts from the embedded Caveat.zip archive
type FontManager struct {
	fonts map[string]*truetype.Font
}

// FlagManager manages loaded flag bitmaps from the embedded w320.zip archive
type FlagManager struct {
	flags map[string]image.Image
}

// Global country lookup instance
var countryLookup *CountryLookup

// init initializes the country lookup data
func init() {
	var err error
	countryLookup, err = loadCountryData()
	if err != nil {
		// Log error but don't panic - application can still work without country lookup
		fmt.Printf("Warning: failed to load country data: %v\n", err)
	}
}

// GameInfoConfig holds configuration for drawing game information
type GameInfoConfig struct {
	BackgroundColor color.RGBA
	TextColor       color.RGBA
	BorderColor     color.RGBA
	FontSize        float64
	Padding         int
	BorderWidth     int
}

// loadCountryData loads and parses the countries CSV data
func loadCountryData() (*CountryLookup, error) {
	// Read the CSV file
	csvData, err := files.ReadFile("countries.csv")
	if err != nil {
		return nil, fmt.Errorf("failed to read countries.csv: %w", err)
	}

	// Parse CSV
	reader := csv.NewReader(strings.NewReader(string(csvData)))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse countries CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("countries CSV has insufficient data")
	}

	// Skip header row and create country data
	countries := make([]Country, 0, len(records)-1)
	nameToAlpha2 := make(map[string]string)
	alpha2ToName := make(map[string]string)

	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) < 11 {
			continue // Skip malformed records
		}

		country := Country{
			Name:                   record[0],
			Alpha2:                 record[1],
			Alpha3:                 record[2],
			CountryCode:            record[3],
			ISO31662:               record[4],
			Region:                 record[5],
			SubRegion:              record[6],
			IntermediateRegion:     record[7],
			RegionCode:             record[8],
			SubRegionCode:          record[9],
			IntermediateRegionCode: record[10],
		}

		countries = append(countries, country)

		// Create mappings for case-insensitive lookup
		nameToAlpha2[strings.ToLower(country.Name)] = country.Alpha2
		alpha2ToName[strings.ToUpper(country.Alpha2)] = country.Name
	}

	return &CountryLookup{
		countries:    countries,
		nameToAlpha2: nameToAlpha2,
		alpha2ToName: alpha2ToName,
	}, nil
}

// GetAlpha2ByName returns the alpha-2 code for a given country name.
// The lookup is case-insensitive.
func GetAlpha2ByName(name string) (string, error) {
	if countryLookup == nil {
		return "", fmt.Errorf("country lookup not initialized")
	}

	alpha2, exists := countryLookup.nameToAlpha2[strings.ToLower(name)]
	if !exists {
		return "", fmt.Errorf("country not found: %s", name)
	}
	return alpha2, nil
}

// GetNameByAlpha2 returns the country name for a given alpha-2 code.
// The lookup is case-insensitive.
func GetNameByAlpha2(alpha2 string) (string, error) {
	if countryLookup == nil {
		return "", fmt.Errorf("country lookup not initialized")
	}

	name, exists := countryLookup.alpha2ToName[strings.ToUpper(alpha2)]
	if !exists {
		return "", fmt.Errorf("country not found with alpha-2 code: %s", alpha2)
	}
	return name, nil
}

// GetAllCountries returns all countries from the CSV data
func GetAllCountries() []Country {
	if countryLookup == nil {
		return nil
	}
	return countryLookup.countries
}

// GetCountryByAlpha2 returns full country information by alpha-2 code
func GetCountryByAlpha2(alpha2 string) (*Country, error) {
	if countryLookup == nil {
		return nil, fmt.Errorf("country lookup not initialized")
	}

	for _, country := range countryLookup.countries {
		if strings.EqualFold(country.Alpha2, alpha2) {
			return &country, nil
		}
	}
	return nil, fmt.Errorf("country not found with alpha-2 code: %s", alpha2)
}

// LoadNaturalEarthData loads and parses the Natural Earth GeoJSON data
func LoadNaturalEarthData() (*NaturalEarthData, error) {
	// Read the GeoJSON file
	jsonData, err := files.ReadFile("naturalearth.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read naturalearth.json: %w", err)
	}

	// Parse GeoJSON
	fc, err := geojson.UnmarshalFeatureCollection(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GeoJSON: %w", err)
	}

	var countries []CountryData
	for _, feature := range fc.Features {
		// Extract country name
		name, _ := feature.Properties["NAME"].(string)
		if name == "" {
			// Try alternative name fields
			if altName, ok := feature.Properties["name"].(string); ok {
				name = altName
			} else if altName, ok := feature.Properties["NAME_EN"].(string); ok {
				name = altName
			}
		}

		// Convert geometry to MultiPolygon
		var multiPoly orb.MultiPolygon
		switch geom := feature.Geometry.(type) {
		case orb.Polygon:
			multiPoly = orb.MultiPolygon{geom}
		case orb.MultiPolygon:
			multiPoly = geom
		default:
			continue // Skip non-polygon geometries
		}

		if name != "" {
			countries = append(countries, CountryData{
				Name:     name,
				Geometry: multiPoly,
			})
		}
	}

	return &NaturalEarthData{Countries: countries}, nil
}

// LoadFonts loads all fonts from the embedded Caveat.zip archive
func LoadFonts() (*FontManager, error) {
	// Read the zip file
	zipData, err := files.ReadFile("Caveat.zip")
	if err != nil {
		return nil, fmt.Errorf("failed to read Caveat.zip: %w", err)
	}

	// Create a zip reader
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create zip reader: %w", err)
	}

	fm := &FontManager{
		fonts: make(map[string]*truetype.Font),
	}

	// Extract and load font files
	for _, file := range zipReader.File {
		// Only process .ttf and .otf files
		if len(file.Name) > 4 {
			ext := file.Name[len(file.Name)-4:]
			if ext == ".ttf" || ext == ".otf" {
				// Open the file in the zip
				rc, err := file.Open()
				if err != nil {
					continue // Skip this file
				}

				// Read the font data
				fontData, err := io.ReadAll(rc)
				closeErr := rc.Close()
				if err != nil {
					continue // Skip this file
				}
				if closeErr != nil {
					continue // Skip this file
				}

				// Parse the font
				font, err := truetype.Parse(fontData)
				if err != nil {
					continue // Skip this file
				}

				// Store the font using base filename as key to prevent directory-prefixed mismatches
				fm.fonts[filepath.Base(file.Name)] = font
			}
		}
	}

	// Also load .ttf files directly from the embed.FS
	entries, err := files.ReadDir(".")
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".ttf") {
				fontData, err := files.ReadFile(entry.Name())
				if err != nil {
					continue
				}

				font, err := truetype.Parse(fontData)
				if err != nil {
					continue
				}

				fm.fonts[entry.Name()] = font
				slog.Debug("Loaded font from embed.FS", "name", entry.Name())
			}
		}
	}

	if len(fm.fonts) == 0 {
		return nil, fmt.Errorf("no valid fonts found")
	}

	return fm, nil
}

// LoadFlags loads all flag bitmaps from the embedded w320.zip archive
func LoadFlags() (*FlagManager, error) {
	// Read the zip file
	zipData, err := files.ReadFile("w320.zip")
	if err != nil {
		return nil, fmt.Errorf("failed to read w320.zip: %w", err)
	}

	// Create a zip reader
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create zip reader: %w", err)
	}

	fm := &FlagManager{
		flags: make(map[string]image.Image),
	}

	// Extract and load flag files
	for _, file := range zipReader.File {
		// Only process .png files
		if len(file.Name) > 4 && strings.ToLower(file.Name[len(file.Name)-4:]) == ".png" {
			// Open the file in the zip
			rc, err := file.Open()
			if err != nil {
				continue // Skip this file
			}

			// Read the image data
			imgData, err := io.ReadAll(rc)
			closeErr := rc.Close()
			if err != nil {
				continue // Skip this file
			}
			if closeErr != nil {
				continue // Skip this file
			}

			// Decode the PNG image
			img, err := png.Decode(bytes.NewReader(imgData))
			if err != nil {
				continue // Skip this file
			}

			// Extract ISO code from filename (e.g., "us.png" -> "US")
			isoCode := strings.ToUpper(strings.TrimSuffix(file.Name, ".png"))
			fm.flags[isoCode] = img
		}
	}

	if len(fm.flags) == 0 {
		return nil, fmt.Errorf("no valid flag images found in w320.zip")
	}

	return fm, nil
}

// GetFlag returns a flag image by ISO code, or nil if not found
func (fm *FlagManager) GetFlag(isoCode string) image.Image {
	if fm == nil {
		return nil
	}
	return fm.flags[strings.ToUpper(isoCode)]
}

// ListFlags returns all available flag ISO codes
func (fm *FlagManager) ListFlags() []string {
	if fm == nil {
		return nil
	}
	var codes []string
	for code := range fm.flags {
		codes = append(codes, code)
	}
	return codes
}

// GetFont returns a font by name, or the first available font if name is empty
func (fm *FontManager) GetFont(name string) *truetype.Font {
	if name != "" {
		if font, exists := fm.fonts[name]; exists {
			return font
		}
	}

	// Return first available font (unpredictable order)
	for _, font := range fm.fonts {
		return font
	}
	return nil
}

// GetUIFont returns a primary UI font (Caveat-Regular) or a suitable fallback,
// ensuring it never accidentally returns an effect font like Matrix-Code.
func (fm *FontManager) GetUIFont() *truetype.Font {
	if fm == nil {
		return nil
	}

	priority := []string{
		"Caveat-Regular.ttf",
		"Caveat-VariableFont_wght.ttf",
		"Caveat-Medium.ttf",
		"Caveat-SemiBold.ttf",
	}

	for _, name := range priority {
		if font, exists := fm.fonts[name]; exists {
			return font
		}
	}

	// If no Caveat fonts found, return any font that is NOT Matrix-style
	for name, font := range fm.fonts {
		if !strings.Contains(strings.ToLower(name), "matrix") {
			return font
		}
	}

	return nil
}

// ListFonts returns all available font names
func (fm *FontManager) ListFonts() []string {
	var names []string
	for name := range fm.fonts {
		names = append(names, name)
	}
	return names
}

// DrawGameInfoRectangle draws a game information rectangle with text using the loaded fonts
func DrawGameInfoRectangle(img *image.RGBA, fm *FontManager, x, y, width, height int, lines []string, config GameInfoConfig) error {
	if fm == nil {
		return fmt.Errorf("font manager is nil")
	}

	// Always try to use UI-suitable fonts for the status rectangle
	ttfFont := fm.GetUIFont()
	if ttfFont == nil {
		return fmt.Errorf("no suitable UI fonts available")
	}

	// Validate rectangle dimensions
	if width <= 0 || height <= 0 {
		return fmt.Errorf("invalid rectangle dimensions: width=%d, height=%d", width, height)
	}

	// Validate text color has proper alpha
	if config.TextColor.A == 0 {
		return fmt.Errorf("text color has zero alpha - text would be invisible")
	}

	// Use ONLY the freetype method for proper filled text rendering
	// The golang.org/x/image/font method renders outlined text by default
	return drawTextWithFreetype(img, ttfFont, x, y, height, lines, config)
}

// drawTextWithFreetype renders text using freetype with proper filled glyphs
func drawTextWithFreetype(img *image.RGBA, ttfFont *truetype.Font, x, y, height int, lines []string, config GameInfoConfig) error {
	// Create font context for filled font rendering
	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(ttfFont)
	c.SetFontSize(config.FontSize)
	c.SetClip(img.Bounds())
	c.SetDst(img)

	// Ensure consistent color values with full opacity for stable rendering
	textColor := config.TextColor
	if textColor.A == 0 {
		textColor.A = 255 // Force full opacity if alpha is 0
	}
	// Normalize text color to prevent variations
	if textColor.A != 255 {
		textColor.A = 255 // Always use full opacity for consistent boldness
	}

	// Create a solid color source for filling the font
	// This is the KEY to getting filled text instead of outlined text
	c.SetSrc(image.NewUniform(textColor))

	// Calculate line height with proper spacing for better text display
	lineHeight := int(config.FontSize * 1.5) // Increased from 1.3 to 1.5 for better line spacing

	// Draw text lines using freetype context for FILLED rendering
	textY := y + config.Padding + int(config.FontSize)
	for _, line := range lines {
		if textY > y+height-config.Padding {
			break // Don't draw outside the rectangle
		}

		textX := x + config.Padding

		// Use freetype.Pt to create the drawing point
		pt := freetype.Pt(textX, textY)

		// Draw the string - this should render FILLED glyphs
		// The freetype library fills the glyphs when using DrawString with a proper source
		_, err := c.DrawString(line, pt)
		if err != nil {
			return fmt.Errorf("failed to draw text line '%s': %w", line, err)
		}

		textY += lineHeight
	}

	return nil
}

// FindCountryAtPoint finds which country contains the given lat/lng point
func (ne *NaturalEarthData) FindCountryAtPoint(lat, lng float64) string {
	point := orb.Point{lng, lat} // orb uses [lng, lat] order

	for _, country := range ne.Countries {
		if planar.MultiPolygonContains(country.Geometry, point) {
			return country.Name
		}
	}

	return "" // Point not found in any country
}

// GetCountryBounds returns the bounding box for a country
func (ne *NaturalEarthData) GetCountryBounds(countryName string) (minLat, maxLat, minLng, maxLng float64, found bool) {
	for _, country := range ne.Countries {
		if country.Name == countryName {
			bound := country.Geometry.Bound()
			return bound.Min[1], bound.Max[1], bound.Min[0], bound.Max[0], true
		}
	}
	return 0, 0, 0, 0, false
}

// RenderNaturalEarthMap creates a map image with country boundaries from Natural Earth data
func RenderNaturalEarthMap(ne *NaturalEarthData, width, height int, black bool, hitCountries map[string]int, targetCountry string, flagManager *FlagManager, fontManager *FontManager, boringCountries map[string]bool, recentHitCountries map[string]bool, liberatedCountries map[string]bool) (image.Image, error) {
	// Debug: show available flags
	if flagManager != nil {
		availableFlags := flagManager.ListFlags()
		slog.Debug("Available flags", "flags", availableFlags)
	}

	// Debug: show boring countries
	if boringCountries != nil {
		slog.Debug("Boring countries", "countries", boringCountries)
	}

	// Create the image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill background with ocean gradient waves
	fillOceanBackground(img, width, height, black)

	// Draw each country
	for _, country := range ne.Countries {
		// Get hit count for this country
		hitCount := 0
		if count, exists := hitCountries[country.Name]; exists {
			hitCount = count
		}

		// Check if this country is boring (>=10 hits) and should use sand/rocks gradient
		isBoring := boringCountries != nil && boringCountries[country.Name]

		// New logic: After first hit, show flag. When boring, show sand/rocks gradient.
		if hitCount >= 1 && hitCount < 10 && flagManager != nil && country.getAlpha2Code() != "" {
			// Show flag for countries with 1-9 hits
			alpha2 := country.getAlpha2Code()
			flag := flagManager.GetFlag(alpha2)
			if flag != nil {
				// Check if this country was recently hit to apply gamma correction
				applyGammaCorrection := recentHitCountries != nil && recentHitCountries[country.Name]

				// Draw country with flag background
				drawCountryWithFlagBackground(img, country.Name, country.Geometry, flag, width, height, applyGammaCorrection)
			} else {
				// Fallback to regular color if no flag found
				fillColor := getCountryHitColor(hitCount)
				drawCountryGeometry(img, country.Name, country.Geometry, fillColor, width, height)
			}
		} else if isBoring && hitCount >= 10 {
			// Show Matrix rain for boring countries (10+ hits)
			if fontManager != nil {
				isLiberated := liberatedCountries != nil && liberatedCountries[country.Name]

				if isLiberated && flagManager != nil && country.getAlpha2Code() != "" {
					// Liberated country: conquered while it was an active target.
					// Show the national flag as background so the country glows with
					// its true identity, then overlay semi-transparent Matrix rain to
					// show the Matrix has been weakened but not fully erased.
					alpha2 := country.getAlpha2Code()
					flag := flagManager.GetFlag(alpha2)
					if flag != nil {
						drawCountryWithFlagBackground(img, country.Name, country.Geometry, flag, width, height, false)
					} else {
						// No flag available — fall back to black so the rain is still visible
						drawCountryGeometry(img, country.Name, country.Geometry, color.RGBA{0, 0, 0, 255}, width, height)
					}
					countrySeed := int64(0)
					for _, char := range country.Name {
						countrySeed += int64(char)
					}
					seed := time.Now().UnixNano()/50000000 + countrySeed
					// Semi-transparent rain (alpha ≈ 160/255 ≈ 63%) so the flag shines through
					DrawMatrixRain(img, country.Name, country.Geometry, fontManager, width, height, seed, 160)
				} else {
					// Normal boring country: black background + fully opaque rain
					drawCountryGeometry(img, country.Name, country.Geometry, color.RGBA{0, 0, 0, 255}, width, height)

					countrySeed := int64(0)
					for _, char := range country.Name {
						countrySeed += int64(char)
					}
					seed := time.Now().UnixNano()/50000000 + countrySeed
					DrawMatrixRain(img, country.Name, country.Geometry, fontManager, width, height, seed, 255)
				}
			} else {
				// Fallback to sand/rocks gradient if font manager not available
				drawCountryWithSandRocksGradient(img, country.Name, country.Geometry, hitCount, width, height)
			}
		} else {
			// Regular country drawing logic for unvisited countries or as fallback
			var fillColor color.RGBA
			if hitCount > 0 {
				fillColor = getCountryHitColor(hitCount)
			} else {
				// Default country color for unvisited countries
				if black {
					fillColor = color.RGBA{60, 60, 60, 255} // Dark gray for dark theme
				} else {
					fillColor = color.RGBA{200, 200, 200, 255} // Light gray for light theme
				}
			}
			drawCountryGeometry(img, country.Name, country.Geometry, fillColor, width, height)
		}

		// Draw red border if this is the target country
		if targetCountry != "" && country.Name == targetCountry {
			drawCountryBorder(img, country.Geometry, color.RGBA{255, 0, 0, 255}, width, height, 2) // Red border, 2px thick
		}
	}

	return img, nil
}

// getCountrySpans returns the cached rasterized span list for a country, computing it on the
// first call for a given (name, width, height). The spans exclude interior ring holes and
// are safe for concurrent readers once stored in the cache.
func getCountrySpans(name string, geom orb.MultiPolygon, width, height int) []spanRun {
	countryMaskCacheMu.Lock()
	if countryMaskCache != nil {
		if e, ok := countryMaskCache[name]; ok && e.width == width && e.height == height {
			countryMaskCacheMu.Unlock()
			return e.spans
		}
	}
	countryMaskCacheMu.Unlock()

	// Compute mask via the existing scanline algorithm (runs once per country per resolution).
	mask := image.NewAlpha(image.Rect(0, 0, width, height))
	for _, polygon := range geom {
		if len(polygon) > 0 {
			fillPolygonAlpha(mask, polygon[0], 255, width, height)
		}
		for i := 1; i < len(polygon); i++ {
			fillPolygonAlpha(mask, polygon[i], 0, width, height)
		}
	}

	// Extract compact horizontal span runs directly from the alpha pixel buffer.
	var spans []spanRun
	pix := mask.Pix
	for y := 0; y < height; y++ {
		row := pix[y*mask.Stride : y*mask.Stride+width]
		x := 0
		for x < width {
			for x < width && row[x] == 0 {
				x++
			}
			if x >= width {
				break
			}
			x1 := x
			for x < width && row[x] > 0 {
				x++
			}
			spans = append(spans, spanRun{y, x1, x - 1})
		}
	}

	entry := &countryMaskEntry{spans: spans, width: width, height: height}
	countryMaskCacheMu.Lock()
	if countryMaskCache == nil {
		countryMaskCache = make(map[string]*countryMaskEntry)
	}
	countryMaskCache[name] = entry
	countryMaskCacheMu.Unlock()
	return spans
}

// spansToAlpha reconstructs an *image.Alpha from cached spans without re-running the
// scanline algorithm. Used by DrawMatrixRain which needs an alpha mask for character clipping.
func spansToAlpha(spans []spanRun, width, height int) *image.Alpha {
	mask := image.NewAlpha(image.Rect(0, 0, width, height))
	for _, s := range spans {
		row := mask.Pix[s.y*mask.Stride:]
		for x := s.x1; x <= s.x2; x++ {
			row[x] = 255
		}
	}
	return mask
}

// fillOceanBackground fills the background with ocean gradient waves.
// The computed pixel buffer is cached and reused when the dimensions and theme
// are unchanged, avoiding O(width×height) math.Sin calls on every frame.
func fillOceanBackground(img *image.RGBA, width, height int, dark bool) {
	oceanCacheMu.Lock()
	if oceanCached != nil &&
		oceanCached.width == width &&
		oceanCached.height == height &&
		oceanCached.dark == dark {
		copy(img.Pix, oceanCached.pix)
		oceanCacheMu.Unlock()
		return
	}
	oceanCacheMu.Unlock()

	// Define ocean colors based on theme
	var deepOcean, shallowOcean, waveHighlight color.RGBA

	if dark {
		// Dark theme ocean colors
		deepOcean = color.RGBA{15, 25, 45, 255}     // Deep dark blue
		shallowOcean = color.RGBA{25, 40, 70, 255}  // Medium dark blue
		waveHighlight = color.RGBA{35, 55, 95, 255} // Lighter dark blue
	} else {
		// Light theme ocean colors
		deepOcean = color.RGBA{65, 105, 180, 255}      // Deep ocean blue
		shallowOcean = color.RGBA{100, 140, 210, 255}  // Medium ocean blue
		waveHighlight = color.RGBA{135, 175, 235, 255} // Light ocean blue
	}

	// Create wave pattern using multiple sine waves
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Calculate multiple wave patterns with different frequencies and phases
			wave1 := math.Sin(float64(x)*0.02+float64(y)*0.015+0) * 0.3
			wave2 := math.Sin(float64(x)*0.035+float64(y)*0.008+math.Pi/3) * 0.2
			wave3 := math.Sin(float64(x)*0.01+float64(y)*0.025+math.Pi/6) * 0.15

			// Combine waves and normalize to 0-1 range
			combinedWave := (wave1 + wave2 + wave3 + 0.65) // Offset to keep mostly positive

			// Add some vertical gradient (deeper at bottom)
			depthGradient := float64(y) / float64(height) * 0.3

			// Combine wave and depth for final intensity
			intensity := math.Max(0, math.Min(1, combinedWave+depthGradient))

			// Interpolate between ocean colors based on intensity
			var finalColor color.RGBA
			if intensity < 0.33 {
				// Deep to shallow interpolation
				t := intensity / 0.33
				finalColor = interpolateColor(deepOcean, shallowOcean, t)
			} else if intensity < 0.66 {
				// Shallow to highlight interpolation
				t := (intensity - 0.33) / 0.33
				finalColor = interpolateColor(shallowOcean, waveHighlight, t)
			} else {
				// Highlight with some variation
				t := (intensity - 0.66) / 0.34
				brighterHighlight := color.RGBA{
					uint8(math.Min(255, float64(waveHighlight.R)+t*20)),
					uint8(math.Min(255, float64(waveHighlight.G)+t*20)),
					uint8(math.Min(255, float64(waveHighlight.B)+t*20)),
					255,
				}
				finalColor = interpolateColor(waveHighlight, brighterHighlight, t)
			}

			img.SetRGBA(x, y, finalColor)
		}
	}

	// Store a copy of the freshly computed pixels in the cache.
	oceanCacheMu.Lock()
	cached := make([]uint8, len(img.Pix))
	copy(cached, img.Pix)
	oceanCached = &oceanCacheEntry{pix: cached, width: width, height: height, dark: dark}
	oceanCacheMu.Unlock()
}

// interpolateColor linearly interpolates between two colors
func interpolateColor(c1, c2 color.RGBA, t float64) color.RGBA {
	// Clamp t to [0, 1]
	t = math.Max(0, math.Min(1, t))

	return color.RGBA{
		R: uint8(float64(c1.R) + t*(float64(c2.R)-float64(c1.R))),
		G: uint8(float64(c1.G) + t*(float64(c2.G)-float64(c1.G))),
		B: uint8(float64(c1.B) + t*(float64(c2.B)-float64(c1.B))),
		A: uint8(float64(c1.A) + t*(float64(c2.A)-float64(c1.A))),
	}
}

// getCountryHitColor returns the color for a country based on hit count
func getCountryHitColor(hitCount int) color.RGBA {
	// This function is no longer used in the new logic, keeping for compatibility
	if hitCount >= 10 {
		// Bright red for occupied countries (conquered)
		return color.RGBA{255, 50, 50, 200}
	}

	// Progressive color intensity based on hit count (1-9)
	intensity := float64(hitCount) / 9.0 // Normalize to 0-1 for hits 1-9

	// Color progression: Light Yellow -> Orange -> Dark Orange
	red := uint8(255)
	green := uint8(255 - intensity*150) // Fade from 255 to 105
	blue := uint8(50 * (1 - intensity)) // Fade from 50 to 0
	alpha := uint8(80 + intensity*100)  // Alpha from 80 to 180

	return color.RGBA{red, green, blue, alpha}
}

// getSandRocksGradientColor returns a gradient color representing sand and rocks for boring countries
func getSandRocksGradientColor(hitCount int, x, y, width, height int) color.RGBA {
	// Define sand and rock colors
	lightSand := color.RGBA{210, 180, 140, 200} // Light sandy beige
	darkSand := color.RGBA{160, 130, 90, 200}   // Darker sand
	lightRock := color.RGBA{120, 100, 80, 220}  // Light gray-brown rock
	darkRock := color.RGBA{80, 65, 50, 240}     // Dark brownish rock

	// Create spatial variation using position
	normalizedX := float64(x) / float64(width)
	normalizedY := float64(y) / float64(height)

	// Create noise-like patterns using sine waves for natural rock/sand distribution
	noiseX := math.Sin(normalizedX*20.0+normalizedY*15.0)*0.5 + 0.5
	noiseY := math.Sin(normalizedY*18.0+normalizedX*12.0)*0.5 + 0.5

	// Combine noise patterns
	rockiness := (noiseX + noiseY) * 0.5

	// Interpolate between sand and rock based on the noise
	var baseColor color.RGBA
	if rockiness < 0.3 {
		// More sandy areas
		t := rockiness / 0.3
		baseColor = interpolateColor(lightSand, darkSand, t)
	} else if rockiness < 0.7 {
		// Mixed sand and rock
		t := (rockiness - 0.3) / 0.4
		baseColor = interpolateColor(darkSand, lightRock, t)
	} else {
		// Rocky areas
		t := (rockiness - 0.7) / 0.3
		baseColor = interpolateColor(lightRock, darkRock, t)
	}

	// Add slight variation based on hit count to show it's been visited many times
	visitIntensity := math.Min(float64(hitCount-10)/20.0, 1.0) // Normalize extra hits beyond 10

	// Darken slightly with more visits to show "wear"
	baseColor.R = uint8(float64(baseColor.R) * (1.0 - visitIntensity*0.2))
	baseColor.G = uint8(float64(baseColor.G) * (1.0 - visitIntensity*0.2))
	baseColor.B = uint8(float64(baseColor.B) * (1.0 - visitIntensity*0.2))

	return baseColor
}

// drawCountryGeometry draws a country's geometry on the image with solid fill.
// It uses the cached span list for the country, avoiding repeated scanline rasterisation.
func drawCountryGeometry(img *image.RGBA, name string, geom orb.MultiPolygon, fillColor color.RGBA, width, height int) {
	spans := getCountrySpans(name, geom, width, height)
	for _, s := range spans {
		row := img.Pix[s.y*img.Stride:]
		for x := s.x1; x <= s.x2; x++ {
			off := x * 4
			row[off] = fillColor.R
			row[off+1] = fillColor.G
			row[off+2] = fillColor.B
			row[off+3] = fillColor.A
		}
	}
}

// drawCountryWithSandRocksGradient draws a country's geometry with sand/rocks gradient pattern.
// Uses the cached span list to avoid repeated scanline rasterisation.
func drawCountryWithSandRocksGradient(img *image.RGBA, name string, geom orb.MultiPolygon, hitCount, width, height int) {
	spans := getCountrySpans(name, geom, width, height)
	for _, s := range spans {
		for x := s.x1; x <= s.x2; x++ {
			gradientColor := getSandRocksGradientColor(hitCount, x, s.y, width, height)
			img.SetRGBA(x, s.y, gradientColor)
		}
	}
}

// applyRandomGammaCorrection applies random gamma correction to a color to indicate recent activity
func applyRandomGammaCorrection(c color.Color) color.Color {
	r, g, b, a := c.RGBA()

	// Generate random gamma value between 0.7 and 1.4 for subtle but noticeable effect
	// Use current time with pixel color for pseudo-randomness that changes per frame
	seed := (time.Now().UnixNano() / 1000000) + int64(r+g+b) // Changes every millisecond
	rng := rand.New(rand.NewSource(seed))
	gamma := 0.7 + rng.Float64()*0.7 // Random gamma between 0.7 and 1.4 (more subtle range)

	// Apply gamma correction
	// Convert from 16-bit to 8-bit, apply gamma, convert back
	rNorm := float64(r>>8) / 255.0
	gNorm := float64(g>>8) / 255.0
	bNorm := float64(b>>8) / 255.0

	rGamma := math.Pow(rNorm, gamma)
	gGamma := math.Pow(gNorm, gamma)
	bGamma := math.Pow(bNorm, gamma)

	// Clamp and convert back to 8-bit
	rFinal := uint8(math.Min(255, math.Max(0, rGamma*255)))
	gFinal := uint8(math.Min(255, math.Max(0, gGamma*255)))
	bFinal := uint8(math.Min(255, math.Max(0, bGamma*255)))
	aFinal := uint8(a >> 8) // Keep original alpha

	return color.RGBA{rFinal, gFinal, bFinal, aFinal}
}

// drawCountryWithFlagBackground draws a country's geometry with a flag image as background.
// If applyGammaCorrection is true, applies random gamma correction to indicate recent activity.
// Uses the cached span list to avoid repeated scanline rasterisation.
func drawCountryWithFlagBackground(img *image.RGBA, name string, geom orb.MultiPolygon, flag image.Image, width, height int, applyGammaCorrection bool) {
	spans := getCountrySpans(name, geom, width, height)

	// Get flag dimensions
	flagBounds := flag.Bounds()
	originalFlagWidth := flagBounds.Dx()
	originalFlagHeight := flagBounds.Dy()

	// Calculate country bounds in pixel coordinates
	countryBound := geom.Bound()
	minX, minY := geoToPixel(countryBound.Max[1], countryBound.Min[1], width, height)
	_, maxY := geoToPixel(countryBound.Min[1], countryBound.Max[0], width, height)

	if minY > maxY {
		minY, maxY = maxY, minY
	}

	countryPixelHeight := int(maxY - minY)
	if countryPixelHeight <= 0 || originalFlagHeight <= 0 {
		return
	}

	scaleFactor := float64(countryPixelHeight) / float64(originalFlagHeight)
	scaledFlagWidth := int(float64(originalFlagWidth) * scaleFactor)
	scaledFlagHeight := countryPixelHeight
	if scaledFlagWidth <= 0 || scaledFlagHeight <= 0 {
		return
	}

	minXi := int(minX)
	minYi := int(minY)
	for _, s := range spans {
		relY := s.y - minYi
		flagY := (relY % scaledFlagHeight) * originalFlagHeight / scaledFlagHeight
		if flagY >= originalFlagHeight {
			flagY = originalFlagHeight - 1
		}
		if flagY < 0 {
			flagY = 0
		}
		for x := s.x1; x <= s.x2; x++ {
			relX := x - minXi
			flagX := (relX % scaledFlagWidth) * originalFlagWidth / scaledFlagWidth
			if flagX >= originalFlagWidth {
				flagX = originalFlagWidth - 1
			}
			if flagX < 0 {
				flagX = 0
			}
			flagColor := flag.At(flagX, flagY)
			if applyGammaCorrection {
				flagColor = applyRandomGammaCorrection(flagColor)
			}
			img.Set(x, s.y, flagColor)
		}
	}
}

// fillPolygonAlpha fills a polygon in an alpha channel using a scanline algorithm
func fillPolygonAlpha(img *image.Alpha, ring orb.Ring, alpha uint8, width, height int) {
	if len(ring) < 3 {
		return // Need at least 3 points for a polygon
	}

	// Convert geographic coordinates to pixel coordinates
	points := make([]image.Point, len(ring))
	for i, coord := range ring {
		x, y := geoToPixel(coord[1], coord[0], width, height) // lat, lng
		points[i] = image.Point{X: int(x), Y: int(y)}
	}

	// Find the bounding box
	minY, maxY := points[0].Y, points[0].Y
	for _, p := range points {
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	// Clamp to image bounds
	if minY < 0 {
		minY = 0
	}
	if maxY >= height {
		maxY = height - 1
	}

	// Scanline fill algorithm
	for y := minY; y <= maxY; y++ {
		intersections := findIntersections(points, y)
		if len(intersections) < 2 {
			continue
		}

		// Sort intersections by x coordinate
		for i := 0; i < len(intersections)-1; i++ {
			for j := i + 1; j < len(intersections); j++ {
				if intersections[i] > intersections[j] {
					intersections[i], intersections[j] = intersections[j], intersections[i]
				}
			}
		}

		// Fill between pairs of intersections
		for i := 0; i < len(intersections)-1; i += 2 {
			x1 := intersections[i]
			x2 := intersections[i+1]

			// Clamp to image bounds
			if x1 < 0 {
				x1 = 0
			}
			if x2 >= width {
				x2 = width - 1
			}

			// Fill the line
			for x := x1; x <= x2; x++ {
				if x >= 0 && x < width && y >= 0 && y < height {
					img.SetAlpha(x, y, color.Alpha{A: alpha})
				}
			}
		}
	}
}

// findIntersections finds all x-intersections of polygon edges with a horizontal line at y
func findIntersections(points []image.Point, y int) []int {
	var intersections []int

	for i := 0; i < len(points); i++ {
		j := (i + 1) % len(points)
		p1, p2 := points[i], points[j]

		// Check if the edge crosses the scanline
		if (p1.Y <= y && p2.Y > y) || (p2.Y <= y && p1.Y > y) {
			// Calculate intersection point
			if p2.Y != p1.Y { // Avoid division by zero
				x := p1.X + (y-p1.Y)*(p2.X-p1.X)/(p2.Y-p1.Y)
				intersections = append(intersections, x)
			}
		}
	}

	return intersections
}

// drawCountryBorder draws the border outline of a country's geometry
func drawCountryBorder(img *image.RGBA, geom orb.MultiPolygon, borderColor color.RGBA, width, height, thickness int) {
	for _, polygon := range geom {
		for _, ring := range polygon {
			// Convert geographic coordinates to pixel coordinates and draw border
			for i := 0; i < len(ring)-1; i++ {
				x1, y1 := geoToPixel(ring[i][1], ring[i][0], width, height)     // lat, lng
				x2, y2 := geoToPixel(ring[i+1][1], ring[i+1][0], width, height) // lat, lng

				// Draw thick line for border
				drawThickLine(img, int(x1), int(y1), int(x2), int(y2), borderColor, thickness)
			}
		}
	}
}

// drawThickLine draws a line with specified thickness
func drawThickLine(img *image.RGBA, x1, y1, x2, y2 int, col color.RGBA, thickness int) {
	// For simplicity, draw multiple parallel lines to create thickness
	for t := -thickness / 2; t <= thickness/2; t++ {
		for s := -thickness / 2; s <= thickness/2; s++ {
			drawLine(img, x1+t, y1+s, x2+t, y2+s, col)
		}
	}
}

// geoToPixel converts geographic coordinates to pixel coordinates
func geoToPixel(lat, lng float64, width, height int) (float64, float64) {
	// Convert longitude (-180 to 180) to x coordinate (0 to width)
	x := (lng + 180) * float64(width) / 360

	// Convert latitude (90 to -90) to y coordinate (0 to height)
	y := (90 - lat) * float64(height) / 180

	return x, y
}

// drawLine draws a simple line between two points
func drawLine(img *image.RGBA, x1, y1, x2, y2 int, col color.RGBA) {
	// Simple Bresenham line algorithm
	dx := int(math.Abs(float64(x2 - x1)))
	dy := int(math.Abs(float64(y2 - y1)))

	var sx, sy int
	if x1 < x2 {
		sx = 1
	} else {
		sx = -1
	}
	if y1 < y2 {
		sy = 1
	} else {
		sy = -1
	}

	err := dx - dy
	x, y := x1, y1

	for {
		if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
			img.Set(x, y, col)
		}

		if x == x2 && y == y2 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x += sx
		}
		if e2 < dx {
			err += dx
			y += sy
		}
	}
}

// DrawMatrixRain draws a Matrix-style falling code effect within a country's geometry
func DrawMatrixRain(img *image.RGBA, name string, geom orb.MultiPolygon, fm *FontManager, width, height int, seed int64, rainAlpha uint8) {
	if fm == nil {
		return
	}

	// Try to get the Matrix font
	font := fm.GetFont("Matrix-Code.ttf")
	if font == nil {
		font = fm.GetFont("") // Fallback to any available font
	}
	if font == nil {
		return
	}

	// Build the clipping mask from cached spans (avoids re-running the scanline algorithm).
	mask := spansToAlpha(getCountrySpans(name, geom, width, height), width, height)

	// Calculate country bounds to limit the area we process
	bound := geom.Bound()
	minX_f, minY_f := geoToPixel(bound.Max[1], bound.Min[0], width, height)
	maxX_f, maxY_f := geoToPixel(bound.Min[1], bound.Max[0], width, height)

	minX, minY := int(minX_f), int(minY_f)
	maxX, maxY := int(maxX_f), int(maxY_f)

	if minX > maxX {
		minX, maxX = maxX, minX
	}
	if minY > maxY {
		minY, maxY = maxY, minY
	}

	// Pad bounds slightly
	minX -= 10
	minY -= 10
	maxX += 10
	maxY += 10

	// Clamp to image bounds
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= width {
		maxX = width - 1
	}
	if maxY >= height {
		maxY = height - 1
	}

	// Matrix rain parameters
	fontSize := 12.0
	charSpacing := 14
	colSpacing := 10

	// Create freetype context
	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(font)
	c.SetFontSize(fontSize)
	c.SetClip(img.Bounds())
	c.SetDst(img)

	// Custom characters for Matrix rain
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789$+-*/=%\"'#&_(),.;:?!"

	rng := rand.New(rand.NewSource(seed))

	// Draw columns of falling code
	for x := minX; x <= maxX; x += colSpacing {
		// Use a stable seed for this column based on X and the overall seed
		colSeed := seed + int64(x*137)
		colRng := rand.New(rand.NewSource(colSeed))

		// Random column properties
		speed := 0.5 + colRng.Float64()*1.5
		offset := colRng.Float64() * float64(maxY-minY)
		raindropY := minY + int(float64(seed)*speed+offset)%(maxY-minY+1)

		// Draw the "streak" (tail)
		streakLen := 10 + colRng.Intn(20)

		for i := 0; i < streakLen; i++ {
			y := raindropY - (i * charSpacing)
			if y < minY {
				// Wrap within country vertical bounds
				y = maxY - ((minY - y) % (maxY - minY + 1))
			}
			if y < minY || y > maxY {
				continue
			}

			// Only draw if inside the country mask
			if mask.AlphaAt(x, y).A < 128 {
				continue
			}

			// Calculate color based on position in streak (0 is head, brightness decreases)
			brightness := 1.0 - (float64(i) / float64(streakLen))
			var charColor color.RGBA
			if i == 0 {
				// Head is white or very light green
				charColor = color.RGBA{200, 255, 200, rainAlpha}
			} else {
				// Tail is varying shades of green
				green := uint8(50 + brightness*205)
				charColor = color.RGBA{0, green, 0, rainAlpha}
			}

			// Pick a random character
			char := chars[rng.Intn(len(chars))]

			// Draw the character
			c.SetSrc(image.NewUniform(charColor))
			pt := freetype.Pt(x, y)
			_, _ = c.DrawString(string(char), pt)
		}
	}
}
