package main

type Config struct {
	Discord struct {
		Token string `json:"token"`
	} `json:"discord"`
}
