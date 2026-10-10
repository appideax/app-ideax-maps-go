package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseZoom(t *testing.T) {
	z, err := ParseZoom("10")
	require.NoError(t, err)
	assert.Equal(t, 10, z)

	_, err = ParseZoom("-1")
	require.Error(t, err)

	_, err = ParseZoom("23")
	require.Error(t, err)

	_, err = ParseZoom("abc")
	require.Error(t, err)
}

func TestParseTileCoord(t *testing.T) {
	x, err := ParseTileCoord(10, "512")
	require.NoError(t, err)
	assert.Equal(t, 512, x)

	_, err = ParseTileCoord(10, "-1")
	require.Error(t, err)

	_, err = ParseTileCoord(1, "2")
	require.Error(t, err)
}

func TestParseLatLon(t *testing.T) {
	lat, lon, err := ParseLatLon("-0.1807", "-78.4678")
	require.NoError(t, err)
	assert.InDelta(t, -0.1807, lat, 0.0001)
	assert.InDelta(t, -78.4678, lon, 0.0001)

	_, _, err = ParseLatLon("91", "0")
	require.Error(t, err)

	_, _, err = ParseLatLon("0", "181")
	require.Error(t, err)

	_, _, err = ParseLatLon("abc", "0")
	require.Error(t, err)
}

func TestParseLimit(t *testing.T) {
	limit, err := ParseLimit("")
	require.NoError(t, err)
	assert.Equal(t, 10, limit)

	limit, err = ParseLimit("50")
	require.NoError(t, err)
	assert.Equal(t, 50, limit)

	_, err = ParseLimit("0")
	require.Error(t, err)

	_, err = ParseLimit("101")
	require.Error(t, err)
}
