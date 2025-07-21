package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/zeebo/bencode"
)

// TorrentInfo represents the parsed torrent file information
type TorrentInfo struct {
	Name        string    `json:"name"`
	InfoHash    string    `json:"info_hash"`
	Size        int64     `json:"size"`
	Files       []File    `json:"files"`
	WebSeeds    []string  `json:"web_seeds"`
	CreatedBy   string    `json:"created_by,omitempty"`
	CreatedDate time.Time `json:"created_date"`
	Comment     string    `json:"comment,omitempty"`
	FilePath    string    `json:"file_path"`
}

// File represents a file within the torrent
type File struct {
	Path   []string `json:"path"`
	Length int64    `json:"length"`
}

// TorrentMetaInfo represents the structure of a .torrent file
type TorrentMetaInfo struct {
	Announce     string      `bencode:"announce"`
	AnnounceList interface{} `bencode:"announce-list,omitempty"`
	Comment      string      `bencode:"comment,omitempty"`
	CreatedBy    string      `bencode:"created by,omitempty"`
	CreationDate int64       `bencode:"creation date,omitempty"`
	Info         InfoDict    `bencode:"info"`
	URLList      interface{} `bencode:"url-list,omitempty"`
}

// InfoDict represents the info dictionary in a torrent file
type InfoDict struct {
	Name        string     `bencode:"name"`
	Length      int64      `bencode:"length,omitempty"`
	Files       []FileDict `bencode:"files,omitempty"`
	PieceLength int64      `bencode:"piece length"`
	Pieces      string     `bencode:"pieces"`
}

// FileDict represents a file in the files list
type FileDict struct {
	Length int64    `bencode:"length"`
	Path   []string `bencode:"path"`
}

// TorznabResponse represents the XML response structure for Torznab API
type TorznabResponse struct {
	XMLName   xml.Name       `xml:"rss"`
	Version   string         `xml:"version,attr"`
	TorznabNS string         `xml:"xmlns:torznab,attr"`
	Channel   TorznabChannel `xml:"channel"`
}

// TorznabChannel represents the channel element in Torznab response
type TorznabChannel struct {
	Title       string        `xml:"title"`
	Description string        `xml:"description"`
	Link        string        `xml:"link"`
	Items       []TorznabItem `xml:"item"`
}

// TorznabItem represents an item in the Torznab response
type TorznabItem struct {
	Title       string           `xml:"title"`
	Description string           `xml:"description"`
	Link        string           `xml:"link"`
	GUID        string           `xml:"guid"`
	PubDate     string           `xml:"pubDate"`
	Size        int64            `xml:"size"`
	Enclosure   TorznabEnclosure `xml:"enclosure"`
	Attributes  []TorznabAttr    `xml:"torznab:attr"`
}

// TorznabEnclosure represents the enclosure element
type TorznabEnclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

// TorznabAttr represents torznab attributes
type TorznabAttr struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// TorrentManager handles torrent operations
type TorrentManager struct {
	torrentsDir string
	torrents    []TorrentInfo
}

// NewTorrentManager creates a new TorrentManager
func NewTorrentManager(torrentsDir string) *TorrentManager {
	return &TorrentManager{
		torrentsDir: torrentsDir,
		torrents:    make([]TorrentInfo, 0),
	}
}

// LoadTorrents scans the torrents directory and loads all torrent files
func (tm *TorrentManager) LoadTorrents() error {
	files, err := ioutil.ReadDir(tm.torrentsDir)
	if err != nil {
		return fmt.Errorf("error reading torrents directory: %v", err)
	}

	tm.torrents = make([]TorrentInfo, 0)

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".torrent") {
			torrentPath := filepath.Join(tm.torrentsDir, file.Name())
			torrentInfo, err := tm.parseTorrentFile(torrentPath)
			if err != nil {
				log.Printf("Error parsing torrent file %s: %v", file.Name(), err)
				continue
			}
			tm.torrents = append(tm.torrents, *torrentInfo)
		}
	}

	log.Printf("Loaded %d torrent files", len(tm.torrents))
	return nil
}

// calculateInfoHash calculates the info hash from torrent data
func calculateInfoHash(torrentData []byte) (string, error) {
	var torrent map[string]interface{}
	err := bencode.DecodeBytes(torrentData, &torrent)
	if err != nil {
		return "", err
	}

	info, ok := torrent["info"]
	if !ok {
		return "", fmt.Errorf("no info section found")
	}

	infoBytes, err := bencode.EncodeBytes(info)
	if err != nil {
		return "", err
	}

	hash := sha1.Sum(infoBytes)
	return hex.EncodeToString(hash[:]), nil
}

// parseTorrentFile parses a .torrent file and extracts information
func (tm *TorrentManager) parseTorrentFile(filePath string) (*TorrentInfo, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	var metaInfo TorrentMetaInfo
	err = bencode.DecodeBytes(data, &metaInfo)
	if err != nil {
		return nil, fmt.Errorf("error decoding bencode: %v", err)
	}

	// Calculate proper info hash
	infoHash, err := calculateInfoHash(data)
	if err != nil {
		log.Printf("Warning: could not calculate info hash for %s: %v", filePath, err)
		infoHash = fmt.Sprintf("%x", data[:20]) // Fallback to simple hash
	}

	var totalSize int64
	var files []File

	if metaInfo.Info.Length > 0 {
		// Single file torrent
		totalSize = metaInfo.Info.Length
		files = []File{{
			Path:   []string{metaInfo.Info.Name},
			Length: metaInfo.Info.Length,
		}}
	} else {
		// Multi-file torrent
		for _, file := range metaInfo.Info.Files {
			totalSize += file.Length
			files = append(files, File{
				Path:   file.Path,
				Length: file.Length,
			})
		}
	}

	createdDate := time.Now()
	if metaInfo.CreationDate > 0 {
		createdDate = time.Unix(metaInfo.CreationDate, 0)
	}

	// Extract web seeds from URLList
	webSeeds := extractWebSeeds(metaInfo.URLList)

	return &TorrentInfo{
		Name:        metaInfo.Info.Name,
		InfoHash:    infoHash,
		Size:        totalSize,
		Files:       files,
		WebSeeds:    webSeeds,
		CreatedBy:   metaInfo.CreatedBy,
		CreatedDate: createdDate,
		Comment:     metaInfo.Comment,
		FilePath:    filePath,
	}, nil
}

// extractWebSeeds extracts web seed URLs from various possible formats
func extractWebSeeds(urlList interface{}) []string {
	if urlList == nil {
		return nil
	}

	var webSeeds []string

	switch v := urlList.(type) {
	case string:
		// Single URL as string
		webSeeds = append(webSeeds, v)
	case []interface{}:
		// List of URLs
		for _, url := range v {
			if urlStr, ok := url.(string); ok {
				webSeeds = append(webSeeds, urlStr)
			}
		}
	case []string:
		// Direct string slice
		webSeeds = v
	}

	return webSeeds
}

// GetTorrents returns all loaded torrents
func (tm *TorrentManager) GetTorrents() []TorrentInfo {
	return tm.torrents
}

// SearchTorrents searches torrents by query
func (tm *TorrentManager) SearchTorrents(query string) []TorrentInfo {
	if query == "" {
		return tm.torrents
	}

	var results []TorrentInfo
	query = strings.ToLower(query)

	for _, torrent := range tm.torrents {
		if strings.Contains(strings.ToLower(torrent.Name), query) {
			results = append(results, torrent)
		}
	}

	return results
}

// APIServer handles HTTP requests
type APIServer struct {
	torrentManager *TorrentManager
	baseURL        string
}

// NewAPIServer creates a new API server
func NewAPIServer(torrentManager *TorrentManager, baseURL string) *APIServer {
	return &APIServer{
		torrentManager: torrentManager,
		baseURL:        baseURL,
	}
}

// handleTorrentsJSON handles JSON API requests
func (s *APIServer) handleTorrentsJSON(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	torrents := s.torrentManager.SearchTorrents(query)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"count":    len(torrents),
		"torrents": torrents,
	})
}

// handleTorznabAPI handles Torznab API requests
func (s *APIServer) handleTorznabAPI(w http.ResponseWriter, r *http.Request) {
	t := r.URL.Query().Get("t")
	query := r.URL.Query().Get("q")

	switch t {
	case "caps":
		s.handleTorznabCaps(w, r)
	case "search":
		s.handleTorznabSearch(w, r, query)
	default:
		s.handleTorznabSearch(w, r, query)
	}
}

// handleTorznabCaps returns capabilities for Torznab API
func (s *APIServer) handleTorznabCaps(w http.ResponseWriter, r *http.Request) {
	capsXML := `<?xml version="1.0" encoding="UTF-8"?>
<caps>
  <server version="1.0" title="WebSeed2Torznab" strapline="Local torrent files with web seeds" email="admin@localhost" url="` + s.baseURL + `" image=""/>
  <limits max="100" default="100"/>
  <registration available="no" open="no"/>
  <searching>
    <search available="yes" supportedParams="q"/>
  </searching>
  <categories>
    <category id="2000" name="Movies"/>
    <category id="5000" name="TV"/>
    <category id="7000" name="Other"/>
  </categories>
</caps>`

	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(capsXML))
}

// handleTorznabSearch handles search requests for Torznab API
func (s *APIServer) handleTorznabSearch(w http.ResponseWriter, r *http.Request, query string) {
	torrents := s.torrentManager.SearchTorrents(query)

	response := TorznabResponse{
		Version:   "2.0",
		TorznabNS: "http://torznab.com/schemas/2015/feed",
		Channel: TorznabChannel{
			Title:       "WebSeed2Torznab",
			Description: "Local torrent files with web seeds",
			Link:        s.baseURL,
			Items:       make([]TorznabItem, 0),
		},
	}

	for _, torrent := range torrents {
		item := TorznabItem{
			Title:       torrent.Name,
			Description: torrent.Comment,
			Link:        fmt.Sprintf("%s/torrent/%s", s.baseURL, url.QueryEscape(filepath.Base(torrent.FilePath))),
			GUID:        torrent.InfoHash,
			PubDate:     torrent.CreatedDate.Format(time.RFC1123Z),
			Size:        torrent.Size,
			Enclosure: TorznabEnclosure{
				URL:    fmt.Sprintf("%s/torrent/%s", s.baseURL, url.QueryEscape(filepath.Base(torrent.FilePath))),
				Length: torrent.Size,
				Type:   "application/x-bittorrent",
			},
			Attributes: []TorznabAttr{
				{Name: "category", Value: "7000"},
				{Name: "size", Value: strconv.FormatInt(torrent.Size, 10)},
				{Name: "seeders", Value: "1"},
				{Name: "peers", Value: "1"},
			},
		}

		if len(torrent.WebSeeds) > 0 {
			item.Attributes = append(item.Attributes, TorznabAttr{
				Name:  "magneturl",
				Value: fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s&ws=%s", torrent.InfoHash, url.QueryEscape(torrent.Name), url.QueryEscape(torrent.WebSeeds[0])),
			})
		}

		response.Channel.Items = append(response.Channel.Items, item)
	}

	w.Header().Set("Content-Type", "application/xml")
	xml.NewEncoder(w).Encode(response)
}

// handleTorrentDownload serves torrent files for download
func (s *APIServer) handleTorrentDownload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	filename := vars["filename"]

	torrentPath := filepath.Join(s.torrentManager.torrentsDir, filename)

	// Check if file exists and is a torrent file
	if _, err := os.Stat(torrentPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	if !strings.HasSuffix(strings.ToLower(filename), ".torrent") {
		http.Error(w, "Invalid file type", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/x-bittorrent")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	http.ServeFile(w, r, torrentPath)
}

// handleRefresh handles refresh requests to reload torrents
func (s *APIServer) handleRefresh(w http.ResponseWriter, r *http.Request) {
	err := s.torrentManager.LoadTorrents()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error refreshing torrents: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Torrents refreshed successfully",
		"count":   len(s.torrentManager.GetTorrents()),
	})
}

func main() {
	torrentsDir := "./torrents"
	if len(os.Args) > 1 {
		torrentsDir = os.Args[1]
	}

	port := "8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	baseURL := fmt.Sprintf("http://localhost:%s", port)
	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		baseURL = envBaseURL
	}

	// Initialize torrent manager
	torrentManager := NewTorrentManager(torrentsDir)
	err := torrentManager.LoadTorrents()
	if err != nil {
		log.Fatalf("Error loading torrents: %v", err)
	}

	// Initialize API server
	apiServer := NewAPIServer(torrentManager, baseURL)

	// Setup routes
	r := mux.NewRouter()

	// JSON API endpoints
	r.HandleFunc("/api/torrents", apiServer.handleTorrentsJSON).Methods("GET")
	r.HandleFunc("/api/refresh", apiServer.handleRefresh).Methods("POST")

	// Torznab API endpoints
	r.HandleFunc("/api/torznab", apiServer.handleTorznabAPI).Methods("GET")

	// Torrent file download
	r.HandleFunc("/torrent/{filename}", apiServer.handleTorrentDownload).Methods("GET")

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}).Methods("GET")

	// Root endpoint with API documentation
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>WebSeed2Torznab API</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        code { background-color: #f4f4f4; padding: 2px 4px; border-radius: 3px; }
        pre { background-color: #f4f4f4; padding: 10px; border-radius: 5px; overflow-x: auto; }
    </style>
</head>
<body>
    <h1>WebSeed2Torznab API</h1>
    <p>A Torznab API for local torrent files with web seed URLs.</p>
    
    <h2>Endpoints</h2>
    <ul>
        <li><strong>GET /api/torrents</strong> - List all torrents in JSON format
            <br><em>Query parameters: ?q=search_term</em></li>
        <li><strong>POST /api/refresh</strong> - Refresh torrent list</li>
        <li><strong>GET /api/torznab</strong> - Torznab API endpoint
            <br><em>Query parameters: ?t=search&q=search_term or ?t=caps</em></li>
        <li><strong>GET /torrent/{filename}</strong> - Download torrent file</li>
        <li><strong>GET /health</strong> - Health check</li>
    </ul>
    
    <h2>Examples</h2>
    <pre>
# Get all torrents as JSON
curl ` + baseURL + `/api/torrents

# Search torrents
curl "` + baseURL + `/api/torrents?q=avengers"

# Torznab capabilities
curl "` + baseURL + `/api/torznab?t=caps"

# Torznab search
curl "` + baseURL + `/api/torznab?t=search&q=cube"

# Refresh torrent list
curl -X POST ` + baseURL + `/api/refresh
    </pre>
    
    <p>Currently serving <strong>` + strconv.Itoa(len(torrentManager.GetTorrents())) + `</strong> torrent files.</p>
</body>
</html>
		`))
	}).Methods("GET")

	log.Printf("Starting WebSeed2Torznab server on port %s", port)
	log.Printf("Serving torrents from: %s", torrentsDir)
	log.Printf("Base URL: %s", baseURL)
	log.Printf("API endpoints:")
	log.Printf("  JSON API: %s/api/torrents", baseURL)
	log.Printf("  Torznab API: %s/api/torznab", baseURL)
	log.Printf("  Torznab Caps: %s/api/torznab?t=caps", baseURL)

	log.Fatal(http.ListenAndServe(":"+port, r))
}
