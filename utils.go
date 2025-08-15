package main

import (
	"encoding/json"
	"os"
)

func NotInSlice[T comparable](target T, slice []T) bool {
	for _, item := range slice {
		if item == target {
			return false
		}
	}
	return true
}

func LoadConfig(file string) (*Config, error) {
	var config Config
	configFile, err := os.Open(file)
	if err != nil {
		return &config, err
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(&config)
	return &config, err
}
