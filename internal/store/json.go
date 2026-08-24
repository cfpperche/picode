package store

import "encoding/json"

func unmarshalJSON(data []byte, v any) error { return json.Unmarshal(data, v) }

func marshalJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
