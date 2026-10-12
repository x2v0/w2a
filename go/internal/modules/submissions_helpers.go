package modules

import (
	"encoding/json"
	"time"
)

func jsonUnmarshal(data []byte, out any) error {
	return json.Unmarshal(data, out)
}

func jsonMarshalIndent(v any) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func sameMap(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}

func nowUTC() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

// orderedOutput returns a map with keys in the desired JSON order.
// Go's json.Marshal orders keys alphabetically; to get schema, generatedAt,
// submissions in that order we use a custom struct.
func orderedOutput(generatedAt string, submissions map[string]any) orderedJSON {
	return orderedJSON{
		Schema:      1,
		GeneratedAt: generatedAt,
		Submissions: submissions,
	}
}

type orderedJSON struct {
	Schema      int            `json:"schema"`
	GeneratedAt string         `json:"generatedAt"`
	Submissions map[string]any `json:"submissions"`
}
