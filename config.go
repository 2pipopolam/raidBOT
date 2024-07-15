package main

import (
    "github.com/pelletier/go-toml"
    "os"
)

type DiscordConfig struct {
    Token      string
    Status     string
    Prefix     string
    PurgeTime  int
    PlayStatus bool
    GuildID    string
    ChannelID  string
}

type YouTubeConfig struct {
    Token string
}

type Config struct {
    Discord DiscordConfig
    YouTube YouTubeConfig
}

func LoadConfig(file string) (*Config, error) {
    f, err := os.Open(file)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    config := &Config{}
    decoder := toml.NewDecoder(f)
    if err := decoder.Decode(config); err != nil {
        return nil, err
    }

    return config, nil
}

