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

	// Генерируем уникальное имя файла
	timestamp := time.Now().UnixNano()
	outputName := fmt.Sprintf("audio_%d", timestamp)
	audioFile, err := ExtractAudio(url, outputName, config.Paths.YoutubeDL)
	if err != nil {
		log.Printf("Ошибка извлечения аудио: %v", err)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Ошибка извлечения аудио: %v", err))
		return
	}

	// Удаляем файл после воспроизведения
	defer func() {
		if err := os.Remove(audioFile); err != nil {
			log.Printf("Ошибка при удалении файла %s: %v", audioFile, err)
		} else {
			log.Printf("Файл удален: %s", audioFile)
		}
	}()

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

	// Удаляем файл после воспроизведения
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

	// Создаем буфер для аудиоданных
	buffer := NewAudioBuffer(bufferSize)

	// Запускаем горутину для чтения файла и заполнения буфера
	go func() {
		file, err := os.Open(audioFile)
		if err != nil {
			log.Printf("не удалось открыть аудиофайл: %v", err)
			return
		}
		defer file.Close()

		encoder, err := gopus.NewEncoder(frameRate, channels, gopus.Audio)
		if err != nil {
			log.Printf("не удалось создать Opus энкодер: %v", err)
			return
		}

		// Улучшенные параметры кодирования
		encoder.SetBitrate(128000)          // Битрейт 128 kbps
		encoder.SetApplication(gopus.Audio) // Оптимизация для музыки

		pcmBuffer := make([]int16, frameSize*channels)
		for {
			err = binary.Read(file, binary.LittleEndian, &pcmBuffer)
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			if err != nil {
				log.Printf("ошибка чтения аудиоданных: %v", err)
				return
			}

			opusFrame, err := encoder.Encode(pcmBuffer, frameSize, frameSize*channels*2)
			if err != nil {
				log.Printf("ошибка кодирования в Opus: %v", err)
				return
			}

			// Ждем, пока в буфере появится место
			for !buffer.Write(opusFrame) {
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	// Ждем начального заполнения буфера (25%)
	time.Sleep(500 * time.Millisecond)

	// Воспроизведение из буфера
	for {
		frame, ok := buffer.Read()
		if !ok {
			time.Sleep(5 * time.Millisecond)
			continue
		}

		select {
		case vc.OpusSend <- frame:
			// Отправлено успешно
		case <-time.After(time.Second):
			return fmt.Errorf("таймаут при отправке аудиоданных")
		}

		// Правильный расчет времени ожидания между кадрами
		sleepDuration := time.Duration(frameSize) * time.Second / time.Duration(frameRate)
		time.Sleep(sleepDuration)
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
