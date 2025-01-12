package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"layeh.com/gopus"
)

const (
	channels   int = 2
	frameRate  int = 48000
	frameSize  int = 960
	bufferSize int = 4096
)

type AudioBuffer struct {
	data     [][]byte
	maxSize  int
	readPos  int
	writePos int
	mu       sync.Mutex
}

func NewAudioBuffer(size int) *AudioBuffer {
	return &AudioBuffer{
		data:     make([][]byte, size),
		maxSize:  size,
		readPos:  0,
		writePos: 0,
	}
}

func (b *AudioBuffer) Write(frame []byte) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	nextWritePos := (b.writePos + 1) % b.maxSize
	if nextWritePos == b.readPos {
		return false // Буфер полон
	}

	b.data[b.writePos] = frame
	b.writePos = nextWritePos
	return true
}

func (b *AudioBuffer) Read() ([]byte, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.readPos == b.writePos {
		return nil, false // Буфер пуст
	}

	frame := b.data[b.readPos]
	b.readPos = (b.readPos + 1) % b.maxSize
	return frame, true
}

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

    // Останавливаем текущее воспроизведение если оно есть
    if playbackControl.IsPlaying() {
        playbackControl.Stop()
        // Удаляем текущий файл если он есть
        currentFile := playbackControl.GetCurrentFile()
        if currentFile != "" {
            if err := os.Remove(currentFile); err != nil {
                log.Printf("Ошибка при удалении старого файла %s: %v", currentFile, err)
            } else {
                log.Printf("Старый файл удален: %s", currentFile)
            }
        }
    }

    timestamp := time.Now().UnixNano()
    outputName := fmt.Sprintf("audio_%d", timestamp)
    audioFile, err := ExtractAudio(url, outputName, config.Paths.YoutubeDL)
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
        // Удаляем файл в случае ошибки
        if err := os.Remove(audioFile); err != nil {
            log.Printf("Ошибка при удалении файла %s: %v", audioFile, err)
        }
        return
    }

    // После завершения воспроизведения
    if err := os.Remove(audioFile); err != nil {
        log.Printf("Ошибка при удалении файла %s: %v", audioFile, err)
    } else {
        log.Printf("Файл удален: %s", audioFile)
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

	defer func() {
		if err := os.Remove(audioFile); err != nil {
			log.Printf("Ошибка при удалении файла %s: %v", audioFile, err)
		} else {
			log.Printf("Файл удален: %s", audioFile)
		}
	}()

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
    // Остановить текущее воспроизведение, если оно есть
    playbackControl.Stop()
    
    // Подождать, чтобы голосовое соединение точно закрылось
    time.Sleep(1 * time.Second)
    
    // Отмечаем начало нового воспроизведения
    playbackControl.Start(audioFile)
    defer playbackControl.Finish()
    
    // Получаем канал остановки
    stopChan := playbackControl.GetStopChan()
    
    // Подключаемся к голосовому каналу
    vc, err := s.ChannelVoiceJoin(config.Discord.GuildID, config.Discord.ChannelID, false, true)
    if err != nil {
        return fmt.Errorf("не удалось присоединиться к голосовому каналу: %w", err)
    }
    
    // Сохраняем соединение в контроллере
    playbackControl.SetVoiceConnection(vc)
    
    err = vc.Speaking(true)
    if err != nil {
        return fmt.Errorf("не удалось включить передачу звука: %w", err)
    }

    // Создаем буфер для аудиоданных
    buffer := NewAudioBuffer(bufferSize)
    
    // Канал для сигнализации о завершении чтения файла
    done := make(chan struct{})
    encodingError := make(chan error, 1)

    // Запускаем горутину для чтения файла и заполнения буфера
    go func() {
        file, err := os.Open(audioFile)
        if err != nil {
            encodingError <- fmt.Errorf("не удалось открыть аудиофайл: %v", err)
            close(done)
            return
        }
        defer file.Close()

        encoder, err := gopus.NewEncoder(frameRate, channels, gopus.Audio)
        if err != nil {
            encodingError <- fmt.Errorf("не удалось создать Opus энкодер: %v", err)
            close(done)
            return
        }

        encoder.SetBitrate(128000)
        encoder.SetApplication(gopus.Audio)

        pcmBuffer := make([]int16, frameSize*channels)
        for {
            select {
            case <-stopChan:
                close(done)
                return
            default:
                err = binary.Read(file, binary.LittleEndian, &pcmBuffer)
                if err == io.EOF || err == io.ErrUnexpectedEOF {
                    close(done)
                    return
                }
                if err != nil {
                    encodingError <- fmt.Errorf("ошибка чтения аудиоданных: %v", err)
                    close(done)
                    return
                }

                opusFrame, err := encoder.Encode(pcmBuffer, frameSize, frameSize*channels*2)
                if err != nil {
                    encodingError <- fmt.Errorf("ошибка кодирования в Opus: %v", err)
                    close(done)
                    return
                }

                for !buffer.Write(opusFrame) {
                    select {
                    case <-stopChan:
                        close(done)
                        return
                    default:
                        time.Sleep(5 * time.Millisecond)
                    }
                }
            }
        }
    }()

    // Ждем начального заполнения буфера
    time.Sleep(500 * time.Millisecond)

    // Воспроизведение из буфера
    for {
        select {
        case err := <-encodingError:
            return err
        case <-stopChan:
            return nil
        case <-done:
            // Проверяем буфер перед выходом
            for {
                frame, ok := buffer.Read()
                if !ok {
                    return nil
                }
                select {
                case vc.OpusSend <- frame:
                case <-stopChan:
                    return nil
                }
            }
        default:
            frame, ok := buffer.Read()
            if !ok {
                select {
                case <-done:
                    return nil
                case <-stopChan:
                    return nil
                default:
                    time.Sleep(5 * time.Millisecond)
                    continue
                }
            }

            select {
            case vc.OpusSend <- frame:
            case <-stopChan:
                return nil
            case <-time.After(time.Second):
                continue // Пропускаем кадр при таймауте
            }
        }
    }
}






func stopAudio(s *discordgo.Session, m *discordgo.MessageCreate, config *Config) {
    playbackControl.Stop()
    
    // Ищем текущее голосовое соединение
    if vs, err := s.State.VoiceState(config.Discord.GuildID, s.State.User.ID); err == nil {
        if vc, err := s.ChannelVoiceJoin(config.Discord.GuildID, vs.ChannelID, false, false); err == nil {
            vc.Speaking(false)
            time.Sleep(100 * time.Millisecond)
            vc.Disconnect()
        }
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

func cleanupOldFiles() {
	patterns := []string{
		"audio_*.pcm",
		"audio_*.wav",
		"vk_audio_*.mp3",
		"audio_*",
	}

	for _, pattern := range patterns {
		files, err := filepath.Glob(pattern)
		if err != nil {
			log.Printf("Ошибка при поиске файлов по шаблону %s: %v", pattern, err)
			continue
		}

		for _, file := range files {
			if err := os.Remove(file); err != nil {
				log.Printf("Ошибка при удалении файла %s: %v", file, err)
			} else {
				log.Printf("Удален старый файл: %s", file)
			}
		}
	}
}
