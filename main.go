package main

import (
    "flag"
    "log"

    "github.com/bwmarrin/discordgo"
)

func main() {
    configFile := flag.String("config", "bot.toml", "path to config file")
    flag.Parse()

    config, err := LoadConfig(*configFile)
    if err != nil {
        log.Fatalf("Error loading config: %v", err)
    }

    dg, err := discordgo.New("Bot " + config.Discord.Token)
    if err != nil {
        log.Fatalf("Error creating Discord session: %v", err)
    }

    dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
        messageCreate(s, m, config)
    })

    err = dg.Open()
    if err != nil {
        log.Fatalf("Error opening Discord session: %v", err)
    }

    log.Println("Bot is now running. Press CTRL+C to exit.")
    select {}
}
