package main

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "strings"

    "github.com/SevereCloud/vksdk/v2/api"
)

type VKAudio struct {
    Artist string `json:"artist"`
    Title  string `json:"title"`
    URL    string `json:"url"`
}

func ExtractAudioFromVK(url string, token string) (string, error) {
    vk := api.NewVK(token)

    // Extract audio ID from URL
    parts := strings.Split(url, "/")
    audioID := parts[len(parts)-1]

    // Split audioID into ownerID and audioID
    idParts := strings.Split(audioID, "_")
    if len(idParts) != 2 {
        return "", fmt.Errorf("invalid VK audio URL format")
    }

    ownerID := idParts[0]
    audioID = idParts[1]

    // Get audio info
    params := api.Params{
        "owner_id":  ownerID,
        "audio_ids": audioID,
    }

    response, err := vk.Request("audio.getById", params)
    if err != nil {
        return "", fmt.Errorf("failed to get audio info: %w", err)
    }

    var audios []VKAudio
    err = json.Unmarshal(response, &audios)
    if err != nil {
        return "", fmt.Errorf("failed to parse audio info: %w", err)
    }

    if len(audios) == 0 {
        return "", fmt.Errorf("audio not found")
    }

    audio := audios[0]

    // Download audio file
    resp, err := http.Get(audio.URL)
    if err != nil {
        return "", fmt.Errorf("failed to download audio: %w", err)
    }
    defer resp.Body.Close()

    outputFile := fmt.Sprintf("%s_%s.mp3", audio.Artist, audio.Title)
    out, err := os.Create(outputFile)
    if err != nil {
        return "", fmt.Errorf("failed to create output file: %w", err)
    }
    defer out.Close()

    _, err = io.Copy(out, resp.Body)
    if err != nil {
        return "", fmt.Errorf("failed to save audio file: %w", err)
    }

    return outputFile, nil
}
