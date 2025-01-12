package main

import (
    "log"
    "os/exec"
)

// ExecuteCommand - Executes a system command
func ExecuteCommand(command string, args ...string) error {
    cmd := exec.Command(command, args...)
    cmd.Stdout = log.Writer()
    cmd.Stderr = log.Writer()
    return cmd.Run()
}
