#!/bin/bash

# GoCommender API Test Script
# Tests the recommendation API with a "Best Of" playlist

set -e  # Exit on any error

API_BASE="http://localhost:8080"
PLAYLIST_NAME="${1:-Best Of}"  # Use first argument or default to "Best Of"
MAX_RESULTS="${2:-5}"          # Use second argument or default to 5

echo "🎵 GoCommender API Test Script"
echo "================================"
echo "Playlist: '$PLAYLIST_NAME'"
echo "Max Results: $MAX_RESULTS"
echo

# Check if server is running
echo "1. Testing API health..."
if ! curl -s -f "$API_BASE/api/health" > /dev/null; then
    echo "❌ Error: GoCommender server is not running on $API_BASE"
    echo "Please start the server with: ./build/gocommender"
    exit 1
fi
echo "✅ Server is running"
echo

# Test health endpoint
echo "2. Getting server health status..."
curl -s "$API_BASE/api/health" | jq .
echo
echo

# Test info endpoint
echo "3. Getting API information..."
curl -s "$API_BASE/api/info" | jq .
echo
echo

# Test Plex connection
echo "4. Testing Plex connection..."
PLEX_RESPONSE=$(curl -s "$API_BASE/api/plex/test")
echo "$PLEX_RESPONSE" | jq .

if echo "$PLEX_RESPONSE" | jq -e '.status == "connected"' > /dev/null; then
    echo "✅ Plex connection successful"
else
    echo "⚠️  Plex connection failed - continuing with test anyway"
fi
echo
echo

# Get available playlists
echo "5. Getting available playlists..."
PLAYLISTS_RESPONSE=$(curl -s "$API_BASE/api/plex/playlists")
echo "$PLAYLISTS_RESPONSE" | jq .

# Check if "Best Of" playlist exists
if echo "$PLAYLISTS_RESPONSE" | jq -e --arg name "$PLAYLIST_NAME" '.playlists[] | select(.name == $name)' > /dev/null; then
    echo "✅ Found '$PLAYLIST_NAME' playlist"
else
    echo "⚠️  '$PLAYLIST_NAME' playlist not found - test will continue anyway"
fi
echo
echo

# Test recommendation endpoint
echo "6. Requesting recommendations for '$PLAYLIST_NAME' playlist..."
REQUEST_PAYLOAD=$(cat <<EOF
{
    "playlist_name": "$PLAYLIST_NAME",
    "max_results": $MAX_RESULTS
}
EOF
)

echo "Request payload:"
echo "$REQUEST_PAYLOAD" | jq .
echo

echo "Making recommendation request..."
RECOMMENDATION_RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d "$REQUEST_PAYLOAD" \
    "$API_BASE/api/recommend")

echo "Response:"
echo "$RECOMMENDATION_RESPONSE" | jq .
echo

# Check if recommendations were successful
if echo "$RECOMMENDATION_RESPONSE" | jq -e '.status == "success"' > /dev/null; then
    SUGGESTION_COUNT=$(echo "$RECOMMENDATION_RESPONSE" | jq '.suggestions | length')
    echo "✅ Successfully received $SUGGESTION_COUNT artist recommendations!"
    
    echo
    echo "📋 Recommended Artists:"
    echo "$RECOMMENDATION_RESPONSE" | jq -r '.suggestions[] | "  • \(.name) (\(.mbid))"'
    
    echo
    echo "📊 Metadata:"
    echo "$RECOMMENDATION_RESPONSE" | jq -r '.metadata | "  • Seed tracks: \(.seed_track_count)\n  • Known artists: \(.known_artist_count)\n  • Processing time: \(.processing_time)\n  • Cache hits: \(.cache_hits)\n  • API calls: \(.api_calls_made)"'
    
elif echo "$RECOMMENDATION_RESPONSE" | jq -e '.error' > /dev/null; then
    ERROR_MSG=$(echo "$RECOMMENDATION_RESPONSE" | jq -r '.error')
    echo "❌ Recommendation failed: $ERROR_MSG"
    exit 1
else
    echo "❌ Unexpected response format"
    exit 1
fi

echo
echo "🎉 API test completed successfully!"
echo
echo "💡 Usage:"
echo "  $0                           # Test with 'Best Of' playlist"
echo "  $0 'My Favorites'            # Test with custom playlist"
echo "  $0 'Rock Classics' 10        # Custom playlist with 10 results"
echo
echo "💡 Tips:"
echo "  • Check server logs for detailed processing information"
echo "  • View API documentation at: $API_BASE/"
echo "  • Ensure Plex server is configured and accessible"