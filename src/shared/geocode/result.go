package geocode

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Result es el formato unificado de geocoding que espera el frontend.
type Result struct {
	ID        string  `json:"id"`
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Source    string  `json:"source,omitempty"`
}

// Response envuelve los resultados con la misma forma que usaban los route
// handlers de Next.js, para no cambiar el contrato del frontend.
type Response struct {
	Results []Result `json:"results"`
}

func formatAddress(parts ...*string) string {
	var clean []string
	seen := make(map[string]struct{})
	for _, p := range parts {
		if p == nil || *p == "" {
			continue
		}
		v := strings.TrimSpace(*p)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		clean = append(clean, v)
	}
	return strings.Join(clean, ", ")
}

func parseCoord(raw any) (float64, float64, bool) {
	coords, ok := raw.([]any)
	if !ok || len(coords) != 2 {
		return 0, 0, false
	}
	lon, ok1 := toFloat(coords[0])
	lat, ok2 := toFloat(coords[1])
	return lat, lon, ok1 && ok2
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case string:
		var f float64
		if _, err := fmt.Sscanf(n, "%f", &f); err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

func stringPtr(props map[string]any, key string) *string {
	if v, ok := props[key].(string); ok && v != "" {
		return &v
	}
	return nil
}

// PhotonTransform convierte la respuesta GeoJSON de Photon al formato Result.
func PhotonTransform(source string) func([]byte) ([]byte, error) {
	return func(body []byte) ([]byte, error) {
		var payload struct {
			Features []struct {
				Geometry   map[string]any `json:"geometry"`
				Properties map[string]any `json:"properties"`
			} `json:"features"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, err
		}

		seen := make(map[string]struct{})
		results := make([]Result, 0, len(payload.Features))

		for _, f := range payload.Features {
			lat, lon, ok := parseCoord(f.Geometry["coordinates"])
			if !ok {
				continue
			}

			props := f.Properties
			id := stableID(props["osm_type"], props["osm_id"], lat, lon)
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}

			name := stringPtr(props, "name")
			street := stringPtr(props, "street")
			housenumber := stringPtr(props, "housenumber")
			city := stringPtr(props, "city")
			state := stringPtr(props, "state")
			country := stringPtr(props, "country")

			var streetWithNumber *string
			if street != nil && housenumber != nil {
				s := *street + " " + *housenumber
				streetWithNumber = &s
			}

			address := formatAddress(name, streetWithNumber, street, city, state, country)
			if address == "" {
				address = fmt.Sprintf("%.5f, %.5f", lat, lon)
			}

			results = append(results, Result{
				ID:        id,
				Address:   address,
				Latitude:  lat,
				Longitude: lon,
				Source:    source,
			})
		}

		return json.Marshal(Response{Results: results})
	}
}

// PeliasTransform convierte la respuesta GeoJSON de Pelias al formato Result.
func PeliasTransform(source string) func([]byte) ([]byte, error) {
	return func(body []byte) ([]byte, error) {
		var payload struct {
			Features []struct {
				Geometry   map[string]any `json:"geometry"`
				Properties map[string]any `json:"properties"`
			} `json:"features"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, err
		}

		seen := make(map[string]struct{})
		results := make([]Result, 0, len(payload.Features))

		for _, f := range payload.Features {
			lat, lon, ok := parseCoord(f.Geometry["coordinates"])
			if !ok {
				continue
			}

			props := f.Properties
			id := stableID("pelias", props["gid"], lat, lon)
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}

			name := stringPtr(props, "name")
			street := stringPtr(props, "street")
			housenumber := stringPtr(props, "housenumber")
			locality := stringPtr(props, "locality")
			region := stringPtr(props, "region")

			var streetWithNumber *string
			if street != nil && housenumber != nil {
				s := *street + " " + *housenumber
				streetWithNumber = &s
			}

			address := formatAddress(name, streetWithNumber, street, locality, region)
			if address == "" {
				address = fmt.Sprintf("%.5f, %.5f", lat, lon)
			}

			results = append(results, Result{
				ID:        id,
				Address:   address,
				Latitude:  lat,
				Longitude: lon,
				Source:    source,
			})
		}

		return json.Marshal(Response{Results: results})
	}
}

func stableID(osmType, osmID any, lat, lon float64) string {
	if t, ok := osmType.(string); ok && t != "" {
		if id, ok := osmID.(float64); ok {
			return fmt.Sprintf("%s%d", t, int64(id))
		}
		if id, ok := osmID.(string); ok && id != "" {
			return fmt.Sprintf("%s%s", t, id)
		}
	}
	return fmt.Sprintf("%.5f,%.5f", lat, lon)
}
