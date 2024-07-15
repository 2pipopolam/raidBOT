package main

import (
    "github.com/bwmarrin/discordgo"
    "os"
    "io"
    "log"
    "time"
)

func PlayPCM(vc *discordgo.VoiceConnection, filePath string) error {
    // Open the file
    file, err := os.Open(filePath)
    if err != nil {
        return err
    }
    defer file.Close()

    // Start sending the audio
    vc.Speaking(true)
    defer vc.Speaking(false)

    buffer := make([]byte, 960*2*2) // Opus encoding uses 20ms frames of stereo audio at 48kHz

    for {
        // Read data into buffer
        n, err := file.Read(buffer)
        if err == io.EOF || err == io.ErrUnexpectedEOF {
            break
        }
        if err != nil {
            return err
        }

        log.Printf("Read %d bytes from PCM file", n)

        // Send buffer to the voice connection
        select {
        case vc.OpusSend <- buffer[:n]:
            log.Printf("Sent %d bytes to OpusSend channel", n)
        case <-time.After(1 * time.Second):
            log.Printf("Timeout sending %d bytes to OpusSend channel", n)
        }
    }

    // Wait for a bit to ensure the audio is played out
    time.Sleep(250 * time.Millisecond)

    return nil
}

