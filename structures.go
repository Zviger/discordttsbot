package main

type Config struct {
	Discord struct {
		Token string `json:"token"`
	} `json:"discord"`
	TTS struct {
		ApiUrl string `json:"apiUrl"`
	} `json:"tts"`
}
