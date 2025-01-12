package main

import (
    "sync"
    "time"
    "github.com/bwmarrin/discordgo"
)

type PlaybackControl struct {
    stopChan chan struct{}
    isPlaying bool
    mu sync.Mutex
    currentFile string
    voiceConnection *discordgo.VoiceConnection
}

var (
    playbackControl = &PlaybackControl{
        stopChan: make(chan struct{}),
    }
)

func (pc *PlaybackControl) Stop() {
    pc.mu.Lock()
    defer pc.mu.Unlock()
    
    if pc.isPlaying {
        close(pc.stopChan)
        pc.stopChan = make(chan struct{})
        pc.isPlaying = false
        
        if pc.voiceConnection != nil {
            pc.voiceConnection.Speaking(false)
            time.Sleep(500 * time.Millisecond)
            pc.voiceConnection.Disconnect()
            pc.voiceConnection = nil
        }
    }
}

func (pc *PlaybackControl) SetVoiceConnection(vc *discordgo.VoiceConnection) {
    pc.mu.Lock()
    defer pc.mu.Unlock()
    pc.voiceConnection = vc
}

func (pc *PlaybackControl) Start(filename string) {
    pc.mu.Lock()
    defer pc.mu.Unlock()
    pc.currentFile = filename
    pc.isPlaying = true
}

func (pc *PlaybackControl) Finish() {
    pc.mu.Lock()
    defer pc.mu.Unlock()
    pc.isPlaying = false
    pc.currentFile = ""
    if pc.voiceConnection != nil {
        pc.voiceConnection.Speaking(false)
        pc.voiceConnection.Disconnect()
        pc.voiceConnection = nil
    }
}

func (pc *PlaybackControl) IsPlaying() bool {
    pc.mu.Lock()
    defer pc.mu.Unlock()
    return pc.isPlaying
}

func (pc *PlaybackControl) GetStopChan() chan struct{} {
    return pc.stopChan
}

func (pc *PlaybackControl) GetCurrentFile() string {
    pc.mu.Lock()
    defer pc.mu.Unlock()
    return pc.currentFile
}
