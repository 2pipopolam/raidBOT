package main

import (
    "log"
)

// CheckError - Checks and logs errors
func CheckError(err error) {
    if err != nil {
        log.Println("Error:", err)
    }
}
