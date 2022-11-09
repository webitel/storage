package google

import "encoding/json"

type Config struct {
	Bucket string `json:"bucket"`
}

func ConfigFromJson(data []byte) Config {
	var c Config
	json.Unmarshal(data, &c)
	return c
}
