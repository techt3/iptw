
# IP Travel Wallpaper (iptw)

IP Travel Wallpaper transforms your network browsing into a gamified world exploration experience. As you visit websites and connect to servers around the globe, your digital footprints are visualized as virtual travels on a beautiful world map that becomes your desktop wallpaper.

## Game Philosophy: Breaking Out of Digital Bubbles

In our interconnected world, most internet traffic flows through a handful of major hosting platforms and CDNs located in just a few countries. This creates digital "bubbles" where we unknowingly consume content from a very limited geographic and cultural perspective.

**IPTW challenges you to:**
- **Escape the hosting monopoly**: Discover websites hosted outside major cloud platforms (AWS, Google Cloud, Azure)
- **Find authentic local voices**: Seek out small community newspapers, local blogs, and regional websites
- **Break news bubbles**: Access diverse perspectives by connecting to servers in different countries
- **Explore digital diversity**: Experience the internet as it was meant to be - truly global and decentralized

The game rewards curiosity and geographic diversity over convenience, encouraging you to venture beyond the mainstream digital highways.


## How It Works: Travel Mechanics

- **Virtual Travel**: Each network connection to a foreign IP address represents a "visit" to that country
- **Progressive Country Coloring**: Countries change appearance based on your visit frequency:
  - **1-9 visits**: Vibrant colors from green to orange (fresh destinations worth exploring)
  - **10+ visits**: Countries become "boring" and turn red (time to find new places!)
- **Exploration Incentives**: 
  - **Target Countries**: Red borders highlight unvisited countries, encouraging global exploration
  - **Achievement System**: Unlock achievements by visiting all countries in geographic regions
  - **Discovery Rewards**: Special recognition for finding rare hosting locations
- **Real-time Visualization**: Watch your virtual travel map expand as you browse, with live connection points
- **Wallpaper Generation**: Your journey becomes a personalized, ever-changing desktop background

## Game Goals & Challenges

### 🎯 **Primary Objectives**
1. **Visit All Countries**: Can you find internet content hosted in every nation?
2. **Regional Completion**: Complete entire continents by discovering local hosting
3. **Rare Country Hunter**: Find websites in countries with minimal global hosting presence
4. **Local News Explorer**: Discover authentic local newspapers and community sites
5. **Escape the Big Three**: Minimize connections to US, EU, and Chinese mega-platforms

### 🌍 **Discovery Challenges**
- **Small Nation Challenge**: Find active websites hosted in microstates and island nations
- **Language Diversity**: Connect to servers hosting content in minority languages
- **Community Voices**: Discover local radio stations, newspapers, and blogs
- **Academic Networks**: Find university and research institution servers worldwide
- **Government Transparency**: Access official government websites hosted locally
- **Cultural Preservation**: Find sites dedicated to local traditions and heritage

### 📰 **Breaking News Bubbles**
The modern internet is dominated by a few major hosting providers, creating invisible geographic barriers to information diversity. IPTW helps you discover:
- **Hyperlocal News**: Small-town newspapers and community bulletins
- **Alternative Perspectives**: Non-Western viewpoints on global events
- **Underrepresented Voices**: Media from developing nations and marginalized communities
- **Direct Sources**: Government, academic, and institutional websites hosted locally
- **Cultural Content**: Local entertainment, art, and cultural preservation sites

## Installation & Usage

### Download Pre-built Binaries

**Latest Release**: Download platform-specific binaries from the [GitHub Releases page](https://github.com/techt3/iptw/releases)

**Platform Support:**
- **macOS**: `iptw-vX.X.X-darwin-amd64.tar.gz` (Intel) / `iptw-vX.X.X-darwin-arm64.tar.gz` (Apple Silicon)
- **Linux**: `iptw-vX.X.X-linux-amd64.tar.gz` (x86_64) / `iptw-vX.X.X-linux-arm64.tar.gz` (ARM64)
- **Windows**: `iptw-vX.X.X-windows-amd64.zip` (x86_64) / `iptw-vX.X.X-windows-arm64.zip` (ARM64)

### Cross-Platform Support
IPTW runs natively on **macOS**, **Linux**, and **Windows** with automatic platform detection for network monitoring.

### Self-Contained Application
- **No Setup Required**: Single executable contains all dependencies
- **No External Downloads**: Everything is embedded in the binary
- **Portable**: Run from any location without installation
- **Privacy-First**: All data processing happens locally on your machine

### Quick Start
1. Download the binary for your platform from [Releases](https://github.com/techt3/iptw/releases)
2. Extract the archive
3. Run `iptw` from terminal/command prompt
4. Start browsing the internet to begin your virtual travels
5. Watch your desktop wallpaper update with your global journey

### Background Service (Recommended)
For continuous automatic operation, install iptw as a background service:

```bash
# Install as background service (auto-starts on boot/login)
./iptw -install-service

# Check service status
./iptw -service-status

# Control service manually  
./iptw -start-service
./iptw -stop-service

# Remove service
./iptw -uninstall-service
```

**Cross-Platform Service Support:**
- **macOS**: LaunchAgent (starts on user login)
- **Linux**: systemd user service (starts on login)  
- **Windows**: Windows Service (starts on system boot)

📖 **For detailed service management, see [SERVICE.md](SERVICE.md)**

## Technical Features

### Embedded Resources (No External Dependencies)
- **GeoIP Database**: IP geolocation powered by embedded GeoLite2-City database
- **World Map Data**: High-quality country boundaries from Natural Earth project
- **Typography**: Custom fonts embedded for beautiful status displays
- **Vector Graphics**: Crisp rendering at any screen resolution
- **Theme Support**: Automatic light/dark theme detection

### Network Monitoring
- **Real-time Connection Tracking**: Monitors all outbound TCP connections
- **Smart Filtering**: Excludes local/private networks, focuses on public internet
- **Protocol Support**: TCP and UDP connection monitoring
- **Performance Optimized**: Efficient native system calls on each platform

### Privacy & Security
- **Local Processing Only**: No data sent to external servers
- **No Account Required**: Completely anonymous usage
- **No Tracking**: Your browsing patterns stay on your device
- **Open Source**: Full transparency in data handling


## Licenses & Attribution

IPTW incorporates several high-quality open resources. We gratefully acknowledge:

### Geographic Data
- **World Map**: `internal/resources/naturalearth.json`
  - **Source**: [Natural Earth](https://www.naturalearthdata.com/)
  - **License**: Public Domain
  - **Description**: High-quality country boundary data at 1:50m scale
  - **Attribution**: Made with Natural Earth, free vector and raster map data from naturalearthdata.com

### GeoIP Database  
- **IP Geolocation**: `internal/geoip/GeoLite2-City.mmdb.zip`
  - **Source**: [MaxMind GeoLite2](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data)
  - **License**: Creative Commons Attribution-ShareAlike 4.0 International License
  - **Description**: Free IP geolocation database
  - **Attribution**: This product includes GeoLite2 data created by MaxMind, available from https://www.maxmind.com

### Typography
- **Font Family**: `internal/resources/Caveat.zip`
  - **Source**: [Google Fonts - Caveat](https://fonts.google.com/specimen/Caveat)
  - **License**: SIL Open Font License (OFL)
  - **Designer**: Pablo Impallari
  - **Description**: Casual handwriting font for friendly, approachable text display

### Additional Resources
- **GeoJSON Maps**: Country boundary processing assisted by [geojson-maps.kyd.au](https://geojson-maps.kyd.au/)

## Contributing

We welcome contributions to help make digital exploration more accessible and diverse! Areas where help is needed:

- **Geographic Coverage**: Help identify websites hosted in underrepresented countries
- **Cultural Insights**: Share knowledge about local internet infrastructure and hosting
- **Language Support**: Internationalization and localization efforts
- **Platform Testing**: Verification across different operating systems
- **Performance Optimization**: Network monitoring efficiency improvements

### Development Setup

**Prerequisites:**
- Go 1.24 or later
- Git
- C compiler (for CGO dependencies)

**Clone and Build:**
```bash
git clone https://github.com/techt3/iptw.git
cd iptw
make build
```

**Development Commands:**
```bash
make help              # Show all available targets
make dev               # Run in development mode
make test              # Run tests
make test-coverage     # Run tests with coverage report
make fmt               # Format code
make lint              # Lint code
make build-all         # Build for all platforms
make package           # Create release packages
make release           # Full release build
```

**Cross-Platform Building:**
The project includes comprehensive cross-platform build support via Makefile and GitHub Actions:

- **Makefile**: Local cross-platform builds for all supported architectures
- **GitHub Actions**: Automated builds and releases on every tag push
- **GoReleaser**: Alternative release automation (optional)

**Supported Build Targets:**
- `darwin/amd64` (macOS Intel)
- `darwin/arm64` (macOS Apple Silicon)
- `linux/amd64` (Linux x86_64)
- `linux/arm64` (Linux ARM64)
- `windows/amd64` (Windows x86_64)
- `windows/arm64` (Windows ARM64)

### Build System

**GitHub Actions Workflows:**
- **`build.yml`**: Main build and release workflow
- **`pr.yml`**: Pull request validation and testing  
- **`deps.yml`**: Automated dependency updates

**Release Process:**
1. Create a new tag: `git tag v1.0.0`
2. Push the tag: `git push origin v1.0.0`
3. GitHub Actions automatically builds and creates a release with binaries for all platforms
4. Release artifacts are available on the GitHub Releases page

### Architecture

**Cross-Platform Service Management:**
- Platform-specific service implementations with Go build tags
- Unified service interface for consistent behavior across platforms
- Native system integration (LaunchAgent, systemd, Windows Service)

**Network Monitoring:**
- Platform-specific network connection tracking
- Real-time IP geolocation with embedded GeoLite2 database
- Efficient connection filtering and processing

## Legal Notice

This software is designed for educational and awareness purposes about global internet infrastructure. Users are responsible for complying with all applicable laws and website terms of service in their jurisdiction. IPTW does not modify network traffic or bypass any security measures - it simply visualizes existing connections. 
