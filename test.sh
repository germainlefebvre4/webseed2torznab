#!/bin/bash

# WebSeed2Torznab Test Script
# Tests all endpoints of the WebSeed2Torznab API

set -e  # Exit on any error

BASE_URL="${BASE_URL:-http://localhost:8080}"
VERBOSE="${VERBOSE:-false}"

echo "🚀 Testing WebSeed2Torznab API at $BASE_URL"
echo "================================================"

# Function to make HTTP requests and check responses
test_endpoint() {
    local method="$1"
    local endpoint="$2"
    local description="$3"
    local expected_code="${4:-200}"
    
    echo -n "Testing $description... "
    
    if [ "$VERBOSE" = "true" ]; then
        echo
        echo "  Request: $method $BASE_URL$endpoint"
    fi
    
    response=$(curl -s -w "\n%{http_code}" -X "$method" "$BASE_URL$endpoint" 2>/dev/null)
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)
    
    if [ "$http_code" = "$expected_code" ]; then
        echo "✅ PASS ($http_code)"
        if [ "$VERBOSE" = "true" ]; then
            echo "  Response: $body" | head -c 200
            echo "..."
        fi
    else
        echo "❌ FAIL (expected $expected_code, got $http_code)"
        if [ "$VERBOSE" = "true" ]; then
            echo "  Response: $body"
        fi
        return 1
    fi
}

# Test health endpoint
test_endpoint "GET" "/health" "Health check"

# Test root endpoint (documentation)
test_endpoint "GET" "/" "Documentation page"

# Test JSON API endpoints
echo
echo "📋 Testing JSON API endpoints"
echo "------------------------------"
test_endpoint "GET" "/api/torrents" "List all torrents (JSON)"
test_endpoint "GET" "/api/torrents?q=cube" "Search torrents (JSON)"
test_endpoint "POST" "/api/refresh" "Refresh torrent list"

# Test Torznab API endpoints
echo
echo "🔍 Testing Torznab API endpoints"
echo "--------------------------------"
test_endpoint "GET" "/api/torznab?t=caps" "Torznab capabilities"
test_endpoint "GET" "/api/torznab?t=search" "Torznab search (all)"
test_endpoint "GET" "/api/torznab?t=search&q=avengers" "Torznab search (query)"

# Test torrent file download (if torrents exist)
echo
echo "📥 Testing torrent file downloads"
echo "--------------------------------"

# Get list of torrents to test download
torrents_json=$(curl -s "$BASE_URL/api/torrents")
torrent_file=$(echo "$torrents_json" | grep -o '"file_path":"[^"]*"' | head -1 | cut -d'"' -f4 | xargs basename 2>/dev/null || echo "")

if [ -n "$torrent_file" ]; then
    test_endpoint "GET" "/torrent/$torrent_file" "Download torrent file"
else
    echo "⚠️  No torrent files found to test download"
fi

# Test invalid endpoints
echo
echo "🚫 Testing error cases"
echo "----------------------"
test_endpoint "GET" "/api/torrents/nonexistent" "Invalid API endpoint" 404
test_endpoint "GET" "/torrent/nonexistent.torrent" "Non-existent torrent" 404

echo
echo "🎉 All tests completed!"

# If verbose, show some sample data
if [ "$VERBOSE" = "true" ]; then
    echo
    echo "📊 Sample API responses"
    echo "======================"
    
    echo "JSON API sample:"
    curl -s "$BASE_URL/api/torrents" | head -c 500
    echo "..."
    
    echo
    echo "Torznab capabilities sample:"
    curl -s "$BASE_URL/api/torznab?t=caps" | head -c 500
    echo "..."
fi
