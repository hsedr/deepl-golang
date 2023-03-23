package deepl

import (
	"fmt"
	"strings"
)

type GlossaryEntries struct {
	Entries map[string]string
}

// NewGlossaryEntries creates a new GlossaryEntries object from a map or a tsv string.
func NewGlossaryEntries(entries interface{}) (*GlossaryEntries, error) {
	g := &GlossaryEntries{
		Entries: make(map[string]string),
	}
	switch entries.(type) {
	case map[string]string:
		g.Entries = entries.(map[string]string)
	case string:
		if tsv, err := g.fromTSV(entries.(string)); err != nil {
			return g, err
		} else {
			g.Entries = tsv
		}
	default:
		return g, fmt.Errorf("invalid type for entries")
	}
	return g, nil
}

func (g *GlossaryEntries) ToTSV() string {
	result := make([]string, 0)
	for k, v := range g.Entries {
		result = append(result, fmt.Sprintf("%s\t%s", k, v))
	}
	return strings.Join(result, "\n")
}

func (g *GlossaryEntries) fromTSV(entries string) (map[string]string, error) {
	result := make(map[string]string)
	for _, entry := range strings.Split(entries, "\n") {
		parts := strings.Split(entry, "\t")
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		} else {
			return result, fmt.Errorf("tab missing in entry: %s", entry)
		}
	}
	return result, nil
}

func (g *GlossaryEntries) Add(source string, target string, overwrite bool) error {
	if !overwrite && g.Entries[source] != "" {
		return fmt.Errorf("entry already exists")
	}
	g.Entries[source] = target
	return nil
}

func (g *GlossaryEntries) validateGlossaryTerm(term string) error {
	if term == "" {
		return fmt.Errorf("term is empty")
	}
	for i, v := range term {
		if (0 <= v && v <= 31) || (128 <= v && v <= 159) || v == 0x2028 || v == 0x2029 {
			return fmt.Errorf("term %s contains invalid character at position %d", term, i)
		}
	}
	return nil
}
