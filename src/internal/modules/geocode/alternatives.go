package geocode

import "strings"

// Palabras que la gente antepone al nombre de un lugar y que el mapa casi nunca
// guarda: "centro comercial Riocentro" figura como "RioCentro Shopping".
var genericWords = map[string]struct{}{
	"cc": {}, "c.c": {}, "c.c.": {}, "mall": {},
	"edificio": {}, "edif": {}, "edif.": {},
}

// alternatives devuelve las variantes del texto que se prueban, en orden, cuando
// la busqueda original no trae nada: sin las palabras genericas, y ademas con
// las dos primeras palabras unidas ("rio centro" figura como "RioCentro").
func alternatives(text string) []string {
	words := strings.Fields(text)
	kept := make([]string, 0, len(words))
	for i := 0; i < len(words); i++ {
		word := strings.ToLower(words[i])
		if word == "centro" && i+1 < len(words) && strings.EqualFold(words[i+1], "comercial") {
			i++
			continue
		}
		if _, generic := genericWords[word]; generic {
			continue
		}
		kept = append(kept, words[i])
	}
	if len(kept) == 0 {
		return nil
	}

	var out []string
	if len(kept) < len(words) {
		out = append(out, strings.Join(kept, " "))
	}
	if len(kept) >= 2 {
		joined := append([]string{kept[0] + kept[1]}, kept[2:]...)
		out = append(out, strings.Join(joined, " "))
	}
	return out
}
