package geocode

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Result es el formato unico de geocodificacion que reciben los clientes, venga
// de Photon o de Pelias.
type Result struct {
	ID        string  `json:"id"`
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Response struct {
	Results []Result `json:"results"`
}

type feature struct {
	Geometry struct {
		Coordinates []float64 `json:"coordinates"`
	} `json:"geometry"`
	Properties struct {
		// Photon
		OSMType string  `json:"osm_type"`
		OSMID   float64 `json:"osm_id"`
		City    string  `json:"city"`
		State   string  `json:"state"`
		// Pelias
		GID      string `json:"gid"`
		Locality string `json:"locality"`
		Region   string `json:"region"`
		// Ambos
		Name        string `json:"name"`
		Street      string `json:"street"`
		HouseNumber string `json:"housenumber"`
	} `json:"properties"`
}

func photonTransform(body []byte) ([]byte, error) {
	return transform(body, func(f *feature) (string, string) {
		id := ""
		if f.Properties.OSMType != "" && f.Properties.OSMID != 0 {
			id = fmt.Sprintf("%s%d", f.Properties.OSMType, int64(f.Properties.OSMID))
		}
		return id, joinAddress(f.Properties.Name, street(f), f.Properties.City, f.Properties.State)
	})
}

func peliasTransform(body []byte) ([]byte, error) {
	return transform(body, func(f *feature) (string, string) {
		id := ""
		if f.Properties.GID != "" {
			// El prefijo evita chocar con los ids de Photon cuando se comparan ambos.
			id = "pelias:" + f.Properties.GID
		}
		return id, joinAddress(f.Properties.Name, street(f), f.Properties.Locality, f.Properties.Region)
	})
}

// transform pasa un GeoJSON de features al formato Result. describe devuelve el
// id estable y la direccion de cada feature; sin ellos se usan las coordenadas.
func transform(body []byte, describe func(*feature) (id, address string)) ([]byte, error) {
	var payload struct {
		Features []feature `json:"features"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(payload.Features))
	results := make([]Result, 0, len(payload.Features))

	for i := range payload.Features {
		f := &payload.Features[i]
		if len(f.Geometry.Coordinates) != 2 {
			continue
		}
		lon, lat := f.Geometry.Coordinates[0], f.Geometry.Coordinates[1]
		coords := fmt.Sprintf("%.5f, %.5f", lat, lon)

		id, address := describe(f)
		if id == "" {
			id = coords
		}
		if address == "" {
			address = coords
		}
		// El proveedor puede repetir features: se queda la primera.
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}

		results = append(results, Result{ID: id, Address: address, Latitude: lat, Longitude: lon})
	}

	return json.Marshal(Response{Results: results})
}

func street(f *feature) string {
	if f.Properties.Street != "" && f.Properties.HouseNumber != "" {
		return f.Properties.Street + " " + f.Properties.HouseNumber
	}
	return f.Properties.Street
}

func joinAddress(parts ...string) string {
	seen := make(map[string]struct{}, len(parts))
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, dup := seen[part]; dup {
			continue
		}
		seen[part] = struct{}{}
		clean = append(clean, part)
	}
	return strings.Join(clean, ", ")
}
