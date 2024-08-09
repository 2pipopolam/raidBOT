package main

import (
    "fmt"
    "log"
    "os"
    "io"
    "strings"
    "time"
    "encoding/binary"

    "github.com/bwmarrin/discordgo"
    "layeh.com/gopus"
)

const (
    channels  int = 2
    frameRate int = 48000
    frameSize int = 960
)

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate, config *Config) {
    if m.Author.ID == s.State.User.ID {
        return
    }

    if strings.HasPrefix(m.Content, config.Discord.Prefix) {
        args := strings.Fields(m.Content)
        command := strings.ToLower(args[0][len(config.Discord.Prefix):])

        switch command {
        case "play":
            if len(args) < 2 {
                s.ChannelMessageSend(m.ChannelID, "Использование: !play [URL]")
                return
            }
            go playAudio(s, m, args[1], config)
        case "playvk":
            if len(args) < 2 {
                s.ChannelMessageSend(m.ChannelID, "Использование: !playvk [URL]")
                return
            }
            go playVKAudio(s, m, args[1], config)
        case "stop":
            stopAudio(s, m, config)
        case "help":
            sendHelpMessage(s, m)
        default:
            s.ChannelMessageSend(m.ChannelID, "Неизвестная команда. Используйте !help для списка команд.")
        }
    }
}

func playAudio(s *discordgo.Session, m *discordgo.MessageCreate, url string, config *Config) {
    log.Printf("Получена команда !play с URL: %s", url)
    s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Начинаю обработку аудио с URL: %s", url))

    audioFile, err := ExtractAudio(url, "audio", config.Paths.YoutubeDL)
    if err != nil {
        log.Printf("Ошибка извлечения аудио: %v", err)
        s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Ошибка извлечения аудио: %v", err))
        return
    }

    s.ChannelMessageSend(m.ChannelID, "Аудио успешно извлечено. Начинаю воспроизведение...")

    err = playAudioFile(s, m, audioFile, config)
    if err != nil {
        log.Printf("Ошибка воспроизведения аудио: %v", err)
        s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Ошибка воспроизведения аудио: %v", err))
        return
    }

    s.ChannelMessageSend(m.ChannelID, "Воспроизведение завершено")
}

func playVKAudio(s *discordgo.Session, m *discordgo.MessageCreate, url string, config *Config) {
    log.Printf("Получена команда !playvk с URL: %s", url)
    s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Начинаю обработку аудио из VK с URL: %s", url))

    audioFile, err := ExtractAudioFromVK(url, config.VK.Token)
    if err != nil {
        log.Printf("Ошибка извлечения аудио из VK: %v", err)
        s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Ошибка извлечения аудио из VK: %v", err))
        return
    }

    s.ChannelMessageSend(m.ChannelID, "Аудио из VK успешно извлечено. Начинаю воспроизведение...")

    err = playAudioFile(s, m, audioFile, config)
    if err != nil {
        log.Printf("Ошибка воспроизведения аудио: %v", err)
        s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Ошибка воспроизведения аудио: %v", err))
        return
    }

    s.ChannelMessageSend(m.ChannelID, "Воспроизведение завершено")
}

func playAudioFile(s *discordgo.Session, m *discordgo.MessageCreate, audioFile string, config *Config) error {
    vc, err := s.ChannelVoiceJoin(config.Discord.GuildID, config.Discord.ChannelID, false, true)
    if err != nil {
        return fmt.Errorf("не удалось присоединиться к голосовому каналу: %w", err)
    }
    defer vc.Disconnect()

    err = vc.Speaking(true)
    if err != nil {
        return fmt.Errorf("не удалось включить передачу звука: %w", err)
    }
    defer vc.Speaking(false)

    file, err := os.Open(audioFile)
    if err != nil {
        return fmt.Errorf("не удалось открыть аудиофайл: %w", err)
    }
    defer file.Close()

    encoder, err := gopus.NewEncoder(frameRate, channels, gopus.Audio)
    if err != nil {
        return fmt.Errorf("не удалось создать Opus энкодер: %w", err)
    }

    buffer := make([]int16, frameSize*channels)
    for {
        err = binary.Read(file, binary.LittleEndian, &buffer)
        if err == io.EOF || err == io.ErrUnexpectedEOF {
            log.Println("Достигнут конец файла")
            return nil
        }
        if err != nil {
            return fmt.Errorf("ошибка чтения аудиоданных: %w", err)
        }

        opus, err := encoder.Encode(buffer, frameSize, frameSize*channels*2)
        if err != nil {
            return fmt.Errorf("ошибка кодирования в Opus: %w", err)
        }

        select {
        case vc.OpusSend <- opus:
            log.Printf("Отправлен Opus фрейм длиной %d байт", len(opus))
        case <-time.After(time.Second):
            return fmt.Errorf("таймаут при отправке аудиоданных")
        }

        time.Sleep(20 * time.Millisecond)
    }
}

func stopAudio(s *discordgo.Session, m *discordgo.MessageCreate, config *Config) {
    vc, err := s.ChannelVoiceJoin(config.Discord.GuildID, config.Discord.ChannelID, false, false)
    if err != nil {
        log.Printf("Ошибка при поиске голосового соединения: %v", err)
        s.ChannelMessageSend(m.ChannelID, "Бот не находится в голосовом канале.")
        return
    }

    err = vc.Disconnect()
    if err != nil {
        log.Printf("Ошибка при отключении от голосового канала: %v", err)
        s.ChannelMessageSend(m.ChannelID, "Произошла ошибка при остановке воспроизведения.")
        return
    }

    s.ChannelMessageSend(m.ChannelID, "Воспроизведение остановлено.")
}

func sendHelpMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
    helpMessage := `Доступные команды:
!play [URL] - воспроизвести аудио с YouTube
!playvk [URL] - воспроизвести аудио из VK
!stop - остановить воспроизведение
!help - показать это сообщение`

    s.ChannelMessageSend(m.ChannelID, helpMessage)
}
