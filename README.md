# WebSeed2Torznab

A Go application that provides both JSON and Torznab APIs for local torrent files containing web seed URLs. This application acts as a local Torznab indexer similar to Jackett, but specifically designed for torrent files with web seeds.

## Features

- **List torrents**: Scans and parses `.torrent` files from a local directory
- **JSON API**: RESTful API for torrent information in JSON format
- **Torznab API**: Compatible with Torznab specification for integration with media servers
- **Web seed support**: Extracts and exposes web seed URLs from torrent files
- **Proper info hash calculation**: Uses SHA-1 hash of the info dictionary
- **File download**: Direct download of torrent files via HTTP
- **Search functionality**: Search torrents by name
- **Real-time refresh**: Ability to reload torrent list without restart

## Installation

### Prerequisites

- Go 1.21 or higher
- Git

### Build from source

```bash
git clone <repository-url>
cd webseed2torznab
go mod tidy
go build -o webseed2torznab main.go
```

## Usage

### Basic usage

```bash
# Run with default settings (./torrents directory, port 8080)
./webseed2torznab

# Specify custom torrents directory
./webseed2torznab /path/to/torrents

# Set custom port via environment variable
PORT=9090 ./webseed2torznab

# Set custom base URL
BASE_URL=http://example.com:8080 ./webseed2torznab
```

### Environment Variables

- `PORT`: Server port (default: 8080)
- `BASE_URL`: Base URL for the server (default: http://localhost:PORT)

## API Endpoints

### JSON API

#### List all torrents
```
GET /api/torrents
```

#### Search torrents
```
GET /api/torrents?q=search_term
```

#### Refresh torrent list
```
GET /api/refresh
```

### Torznab API

#### Get capabilities
```
GET /api/torznab?t=caps
```

#### Search torrents
```
GET /api/torznab?t=search&q=search_term
```

### File Downloads

#### Download torrent file
```
GET /torrent/{filename}
```

### Health Check

#### Service status
```
GET /health
```

## API Examples

### JSON API Examples

```bash
# Get all torrents
curl http://localhost:8080/api/torrents

# Search for torrents containing "avengers"
curl "http://localhost:8080/api/torrents?q=avengers"

# Refresh torrent list
curl -X POST http://localhost:8080/api/refresh
```

### Torznab API Examples

```bash
# Get Torznab capabilities
curl "http://localhost:8080/api/torznab?t=caps"

# Search via Torznab
curl "http://localhost:8080/api/torznab?t=search&q=cube"

# Download a torrent file
curl -O "http://localhost:8080/torrent/Movie.2023.720p.WEBRip.x264.torrent"
```

## Response Formats

### JSON API Response

```json
{
  "status": "success",
  "count": 5,
  "torrents": [
    {
      "name": "Movie Title",
      "info_hash": "72bdf5bd3ed8309db13e72983cc4a3acd4868d91",
      "size": 2530433842,
      "files": [
        {
          "path": ["Movie", "Movie.mkv"],
          "length": 2530433842
        }
      ],
      "web_seeds": [
        "http://example.com/path/to/file.mkv"
      ],
      "created_by": "mktorrent 1.1",
      "created_date": "2025-06-08T00:38:35+02:00",
      "comment": "Sample torrent",
      "file_path": "torrents/Movie.torrent"
    }
  ]
}
```

### Torznab XML Response

```xml
<rss version="2.0">
  <channel>
    <title>WebSeed2Torznab</title>
    <description>Local torrent files with web seeds</description>
    <link>http://localhost:8080</link>
    <item>
      <title>Movie Title</title>
      <description>Sample torrent</description>
      <link>http://localhost:8080/torrent/Movie.torrent</link>
      <guid>72bdf5bd3ed8309db13e72983cc4a3acd4868d91</guid>
      <pubDate>Sun, 08 Jun 2025 00:38:35 +0200</pubDate>
      <size>2530433842</size>
      <enclosure url="http://localhost:8080/torrent/Movie.torrent" length="2530433842" type="application/x-bittorrent"/>
      <torznab:attr name="category" value="7000"/>
      <torznab:attr name="size" value="2530433842"/>
      <torznab:attr name="magneturl" value="magnet:?xt=urn:btih:72bdf5bd3ed8309db13e72983cc4a3acd4868d91&dn=Movie%20Title&ws=http%3A%2F%2Fexample.com%2Fpath%2Fto%2Ffile.mkv"/>
    </item>
  </channel>
</rss>
```

## Integration with Media Servers

### Sonarr/Radarr Integration

1. Add a new Torznab indexer in your media server
2. Set the URL to: `http://your-server:8080/api/torznab`
3. No API key required
4. Test the connection

### Supported Categories

- 2000: Movies
- 5000: TV Shows  
- 7000: Other

## Directory Structure

```
webseed2torznab/
├── main.go           # Main application
├── go.mod            # Go module dependencies
├── go.sum            # Go module checksums
├── torrents/         # Directory containing .torrent files
│   ├── Movie1.torrent
│   ├── Movie2.torrent
│   └── ...
└── README.md         # This file
```

## Features Details

### Torrent Parsing

The application automatically:
- Parses `.torrent` files using bencode
- Calculates proper SHA-1 info hashes
- Extracts file information (name, size, file list)
- Identifies web seed URLs from `url-list` field
- Handles both single-file and multi-file torrents

### Web Seed Support

The application specifically looks for and extracts web seed URLs from the `url-list` field in torrent files. These URLs are:
- Included in JSON responses as `web_seeds` array
- Added to magnet links in Torznab responses as `ws` parameters
- Used to enable HTTP/HTTPS downloading alongside BitTorrent

### Error Handling

- Graceful handling of corrupted torrent files
- Detailed logging of parsing errors
- Continues operation even if some torrents fail to parse
- Returns appropriate HTTP status codes

## Troubleshooting

### Common Issues

1. **No torrents loaded**: Ensure `.torrent` files exist in the torrents directory
2. **Parsing errors**: Check that torrent files are valid and not corrupted
3. **Port conflicts**: Change the PORT environment variable if 8080 is in use
4. **Permission errors**: Ensure the application has read access to the torrents directory

### Logging

The application logs important events including:
- Number of torrents loaded on startup
- Parsing errors for individual torrent files
- Server startup information with endpoints

## License

This project is provided as-is for educational and personal use.

## Contributing

Feel free to submit issues and enhancement requests.
