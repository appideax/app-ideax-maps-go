package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseZoom(t *testing.T) {
	z, err := ParseZoom("10")
	assert.NoError(t, err)
	assert.Equal(t, 10, z)

	for _, raw := range []string{"-1", "23", "abc", ""} {
		_, err = ParseZoom(raw)
		assert.Error(t, err, raw)
	}
}

func TestParseTileCoord(t *testing.T) {
	x, err := ParseTileCoord(2, "3")
	assert.NoError(t, err)
	assert.Equal(t, 3, x)

	for _, raw := range []string{"4", "-1", "abc"} {
		_, err = ParseTileCoord(2, raw)
		assert.Error(t, err, raw)
	}
}

func TestParseLatLng(t *testing.T) {
	lat, lng, err := ParseLatLng("-0.1807", "-78.4678")
	assert.NoError(t, err)
	assert.InDelta(t, -0.1807, lat, 0.0001)
	assert.InDelta(t, -78.4678, lng, 0.0001)

	for _, pair := range [][2]string{{"91", "0"}, {"0", "181"}, {"", "0"}, {"0", "abc"}, {"NaN", "0"}, {"0", "Inf"}} {
		_, _, err = ParseLatLng(pair[0], pair[1])
		assert.Error(t, err, pair)
	}
}

func TestParseBBox(t *testing.T) {
	box, ok := ParseBBox("-78.6, -0.4, -78.3, 0")
	assert.True(t, ok)
	assert.Equal(t, BBox{MinLon: -78.6, MinLat: -0.4, MaxLon: -78.3, MaxLat: 0}, box)
	assert.Equal(t, "-78.6,-0.4,-78.3,0", box.String())

	for _, raw := range []string{"", "1,2,3", "a,b,c,d", "-78.3,-0.4,-78.6,0", "-78.6,0,-78.3,-0.4", "-200,-0.4,-78.3,0", "NaN,0,1,1"} {
		_, ok = ParseBBox(raw)
		assert.False(t, ok, raw)
	}
}
