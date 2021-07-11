package metadata

import "encoding/json"

// Data is the metadata of a web page.
type Data struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// JSON marshals Data into a JSON string.
func (d Data) JSON() (string, error) {
	b, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
