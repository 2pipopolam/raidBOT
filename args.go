package main

import (
    "flag"
)

// Args - Arguments structure
type Args struct {
    Token     string
    GuildID   string
    ChannelID string
}

func parseArgs() Args {
    token := flag.String("token", "", "Bot Token")
    guildID := flag.String("guildID", "", "Guild ID")
    channelID := flag.String("channelID", "", "Channel ID")

    flag.Parse()

    if *token == "" || *guildID == "" || *channelID == "" {
        flag.Usage()
        panic("Required arguments are missing")
    }

    return Args{
        Token:     *token,
        GuildID:   *guildID,
        ChannelID: *channelID,
    }
}

