// Package resources provides embedded static resources for the IP Travel Wallpaper application
package resources

import (
	"archive/zip"
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"math"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/planar"
)

//go:embed *.json *.zip
var files embed.FS

// CountryData represents a country with its geometry and metadata
type CountryData struct {
	Name     string
	Code     string
	Geometry orb.MultiPolygon
}

// NaturalEarthData holds all country data
type NaturalEarthData struct {
	Countries []CountryData
}

// FontManager manages loaded fonts from the embedded Caveat.zip archive
type FontManager struct {
	fonts map[string]*truetype.Font
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

		// Extract country code
		code, _ := feature.Properties["ISO_A3"].(string)
		if code == "" {
			// Try alternative code fields
			if altCode, ok := feature.Properties["iso_a3"].(string); ok {
				code = altCode
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
				Code:     code,
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

				// Store the font using filename as key
				fm.fonts[file.Name] = font
			}
		}
	}

	if len(fm.fonts) == 0 {
		return nil, fmt.Errorf("no valid fonts found in Caveat.zip")
	}

	return fm, nil
}

// GetFont returns a font by name, or the first available font if name is empty
func (fm *FontManager) GetFont(name string) *truetype.Font {
	if name != "" {
		if font, exists := fm.fonts[name]; exists {
			return font
		}
	}

	// Return first available font
	for _, font := range fm.fonts {
		return font
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

	// Always try to use Regular weight for consistency
	ttfFont := fm.GetFont("Caveat-Regular.ttf")
	if ttfFont == nil {
		// Fallback to variable font if regular not available
		ttfFont = fm.GetFont("Caveat-VariableFont_wght.ttf")
	}
	if ttfFont == nil {
		// Final fallback to any available font
		ttfFont = fm.GetFont("")
	}
	if ttfFont == nil {
		return fmt.Errorf("no fonts available")
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
func RenderNaturalEarthMap(ne *NaturalEarthData, width, height int, black bool, hitCountries map[string]int, targetCountry string) (image.Image, error) {
	// Create the image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Background color
	bgColor := color.RGBA{240, 240, 240, 255} // Light background
	if black {
		bgColor = color.RGBA{32, 32, 32, 255} // Dark background
	}

	// Fill background
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// Draw each country
	for _, country := range ne.Countries {
		// Determine country color based on hit count
		var fillColor color.RGBA
		if hitCount, exists := hitCountries[country.Name]; exists && hitCount > 0 {
			fillColor = getCountryHitColor(hitCount)
		} else {
			// Default country color
			if black {
				fillColor = color.RGBA{60, 60, 60, 255} // Dark gray for dark theme
			} else {
				fillColor = color.RGBA{200, 200, 200, 255} // Light gray for light theme
			}
		}

		// Draw country geometry
		drawCountryGeometry(img, country.Geometry, fillColor, width, height)

		// Draw red border if this is the target country
		if targetCountry != "" && country.Name == targetCountry {
			drawCountryBorder(img, country.Geometry, color.RGBA{255, 0, 0, 255}, width, height, 2) // Red border, 2px thick
		}
	}

	return img, nil
}

// getCountryHitColor returns the color for a country based on hit count
func getCountryHitColor(hitCount int) color.RGBA {
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

// drawCountryGeometry draws a country's geometry on the image with solid fill
func drawCountryGeometry(img *image.RGBA, geom orb.MultiPolygon, fillColor color.RGBA, width, height int) {
	for _, polygon := range geom {
		// Fill the main polygon (exterior ring)
		if len(polygon) > 0 {
			fillPolygon(img, polygon[0], fillColor, width, height)
		}

		// Draw holes (interior rings) in background color
		// This creates the proper polygon with holes
		for i := 1; i < len(polygon); i++ {
			// Use transparent color for holes
			holeColor := color.RGBA{0, 0, 0, 0} // Transparent
			fillPolygon(img, polygon[i], holeColor, width, height)
		}
	}
}

// fillPolygon fills a polygon using a scanline algorithm
func fillPolygon(img *image.RGBA, ring orb.Ring, fillColor color.RGBA, width, height int) {
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
					img.Set(x, y, fillColor)
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
