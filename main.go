package main

import (
	"discordttsbot/config"
	"discordttsbot/discord"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	bot, err := discord.NewBot(cfg)

	err = bot.Start()
	if err != nil {
		log.Fatal("Error starting bot:", err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("TTS is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	bot.Stop()
}
