package main

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
)

func ExtractAudio(url, output string, ytdlpPath string) (string, error) {
    wavFile := output + ".wav"
    cmd := exec.Command(ytdlpPath, 
        "-x",
        "--audio-format", "wav",
        "--audio-quality", "0",
        "-o", wavFile,
        url)
    out, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("failed to extract audio: %w, output: %s", err, out)
    }
    
    // Проверяем, существует ли файл
    if _, err := os.Stat(wavFile); os.IsNotExist(err) {
        // Если файл не существует, ищем файл с расширением .wav
        matches, err := filepath.Glob(output + "*.wav")
        if err != nil {
            return "", fmt.Errorf("failed to find output file: %w", err)
        }
        if len(matches) == 0 {
            return "", fmt.Errorf("output file not found")
        }
        wavFile = matches[0]
    }
    
    // Конвертируем WAV в PCM
    pcmFile := output + ".pcm"
    cmd = exec.Command("ffmpeg", 
        "-i", wavFile,
        "-f", "s16le",
        "-acodec", "pcm_s16le",
        "-ar", "48000",
        "-ac", "2",
        pcmFile)
    out, err = cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("failed to convert to PCM: %w, output: %s", err, out)
    }

    // Удаляем временный WAV файл
    os.Remove(wavFile)

    return pcmFile, nil
}
