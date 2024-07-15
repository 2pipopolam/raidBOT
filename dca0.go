package main

import (
    "bufio"
    "io"
    "os"
)

// EncodePCM - Encodes PCM data to DCA format
func EncodePCM(inputFile, outputFile string) error {
    inFile, err := os.Open(inputFile)
    if err != nil {
        return err
    }
    defer inFile.Close()

    outFile, err := os.Create(outputFile)
    if err != nil {
        return err
    }
    defer outFile.Close()

    writer := bufio.NewWriter(outFile)

    // DCA Header
    header := []byte{
        0x44, 0x43, 0x41, 0x30, // DCA0
    }
    writer.Write(header)

    // PCM data
    reader := bufio.NewReader(inFile)
    for {
        sample, err := reader.ReadBytes('\n')
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }
        writer.Write(sample)
    }

    writer.Flush()
    return nil
}

