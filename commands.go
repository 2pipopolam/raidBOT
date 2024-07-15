package main

import (
    "bufio"
    "fmt"
    "log"
    "os"
    "strings"

    "github.com/bwmarrin/discordgo"
)

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate, config *Config) {
    // Ignore all messages created by the bot itself
    if m.Author.ID == s.State.User.ID {
        return
    }

    if strings.HasPrefix(m.Content, "!play") {
        url := strings.TrimSpace(strings.TrimPrefix(m.Content, "!play"))
        log.Printf("Received !play command with URL: %s", url)

        go func() {
            log.Println("Extracting audio...")
            audioFile, err := ExtractAudio(url, "audio.wav")
            if err != nil {
                log.Printf("Error extracting audio: %v", err)
                return
            }

            log.Println("Playing audio...")
            err = playAudioFile(s, m, audioFile, config)
            if err != nil {
                log.Printf("Error playing audio: %v", err)
                return
            }
        }()
    }
}

func playAudioFile(s *discordgo.Session, m *discordgo.MessageCreate, audioFile string, config *Config) error {
    vs, err := s.ChannelVoiceJoin(config.Discord.GuildID, config.Discord.ChannelID, false, true)
    if err != nil {
        return fmt.Errorf("failed to join voice channel: %w", err)
    }
    defer vs.Disconnect()

    vs.Speaking(true)
    defer vs.Speaking(false)

    file, err := os.Open(audioFile)
    if err != nil {
        return fmt.Errorf("failed to open audio file: %w", err)
    }
    defer file.Close()

    reader := bufio.NewReader(file)
    buffer := make([]byte, 3840)
    for {
        n, err := reader.Read(buffer)
        if err != nil {
            break
        }
        vs.OpusSend <- buffer[:n]
    }

    return nil
}
