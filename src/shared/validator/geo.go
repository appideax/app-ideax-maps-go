package validator

import (
	"fmt"
	"strconv"
)

const (
	MaxZoom  = 22
	MaxLimit = 100
)

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

func ParseLatLon(lat, lon string) (float64, float64, error) {
	latitude, err := strconv.ParseFloat(lat, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid latitude: %s", lat)
	}
	longitude, err := strconv.ParseFloat(lon, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid longitude: %s", lon)
	}
	if latitude < -90 || latitude > 90 {
		return 0, 0, fmt.Errorf("latitude out of range: %s", lat)
	}
	if longitude < -180 || longitude > 180 {
		return 0, 0, fmt.Errorf("longitude out of range: %s", lon)
	}
	return latitude, longitude, nil
}

func ParseLimit(limit string) (int, error) {
	if limit == "" {
		return 10, nil
	}
	n, err := strconv.Atoi(limit)
	if err != nil || n < 1 || n > MaxLimit {
		return 0, fmt.Errorf("invalid limit: %s", limit)
	}
	return n, nil
}
