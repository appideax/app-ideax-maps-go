package validator

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const MaxZoom = 22

func ParseZoom(z string) (int, error) {
	zoom, err := strconv.Atoi(z)
	if err != nil || zoom < 0 || zoom > MaxZoom {
		return 0, fmt.Errorf("invalid zoom level: %s", z)
	}
	return zoom, nil
}

func ParseTileCoord(zoom int, coord string) (int, error) {
	value, err := strconv.Atoi(coord)
	if err != nil {
		return 0, fmt.Errorf("invalid tile coordinate: %s", coord)
	}
	max := (1 << zoom) - 1
	if value < 0 || value > max {
		return 0, fmt.Errorf("tile coordinate out of range for zoom %d: %s", zoom, coord)
	}
	return value, nil
}

func ParseLatLng(lat, lng string) (float64, float64, error) {
	latitude, err := parseFinite(lat)
	if err != nil || latitude < -90 || latitude > 90 {
		return 0, 0, fmt.Errorf("invalid latitude: %s", lat)
	}
	longitude, err := parseFinite(lng)
	if err != nil || longitude < -180 || longitude > 180 {
		return 0, 0, fmt.Errorf("invalid longitude: %s", lng)
	}
	return latitude, longitude, nil
}

// BBox es un rectangulo de busqueda: minLon,minLat,maxLon,maxLat.
type BBox struct {
	MinLon, MinLat, MaxLon, MaxLat float64
}

func (b BBox) String() string {
	return fmt.Sprintf("%g,%g,%g,%g", b.MinLon, b.MinLat, b.MaxLon, b.MaxLat)
}

// ParseBBox lee "minLon,minLat,maxLon,maxLat". Devuelve false si viene vacio o
// invalido, y quien llama decide con que rectangulo se queda.
func ParseBBox(raw string) (BBox, bool) {
	parts := strings.Split(raw, ",")
	if len(parts) != 4 {
		return BBox{}, false
	}
	values := make([]float64, 4)
	for i, part := range parts {
		value, err := parseFinite(strings.TrimSpace(part))
		if err != nil {
			return BBox{}, false
		}
		values[i] = value
	}
	box := BBox{MinLon: values[0], MinLat: values[1], MaxLon: values[2], MaxLat: values[3]}
	if box.MinLon >= box.MaxLon || box.MinLat >= box.MaxLat {
		return BBox{}, false
	}
	if box.MinLon < -180 || box.MaxLon > 180 || box.MinLat < -90 || box.MaxLat > 90 {
		return BBox{}, false
	}
	return box, true
}

func parseFinite(raw string) (float64, error) {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("invalid number: %s", raw)
	}
	return value, nil
}
