package main

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

func main() {

	cleanupOldFiles()

	config, err := LoadConfig("bot.toml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	log.Printf("Discord Token: %s", config.Discord.Token[:10]+"...")
	log.Printf("Guild ID: %s", config.Discord.GuildID)
	log.Printf("Channel ID: %s", config.Discord.ChannelID)
	log.Printf("VK Token: %s", config.VK.Token[:10]+"...")
	log.Printf("YouTube-DL Path: %s", config.Paths.YoutubeDL)
	log.Printf("FFmpeg Path: %s", config.Paths.FFmpeg)

	dg, err := discordgo.New("Bot " + config.Discord.Token)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates

	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		messageCreate(s, m, config)
	})

	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	err = dg.Open()
	if err != nil {
		log.Fatalf("Error opening Discord session: %v", err)
	}

	log.Println("Bot is now running. Press CTRL+C to exit.")
	select {}
}
