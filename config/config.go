package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Discord struct {
		Token string `json:"token"`
	} `json:"discord"`
	TTS struct {
		ApiUrl string `json:"apiUrl"`
	} `json:"tts"`
}

func Load(file string) (*Config, error) {
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
