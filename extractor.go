package main

import (
    "fmt"
    "os/exec"
)

func ExtractAudio(url, output string) (string, error) {
    cmd := exec.Command("yt-dlp", "--extract-audio", "--audio-format", "wav", "--output", output, url)
    out, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("failed to extract audio: %w, output: %s", err, out)
    }
    return output, nil
}

