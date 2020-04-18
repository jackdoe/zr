package util

import (
	"encoding/json"
	"io/ioutil"
	"path"
)

type Config struct {
	Kind []string `json:"kind"`
}

func ReadConfig() *Config {
	b, err := ioutil.ReadFile(path.Join(GetDefaultRoot(), "config.json"))
	if err == nil {
		c := &Config{}
		err = json.Unmarshal(b, c)
		if err == nil {
			return c
		}
	}

	return &Config{
		Kind: []string{"man", "so", "su"},
	}
}
