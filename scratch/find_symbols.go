package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	err := filepath.Walk("../pkg/qtui", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			cleanLine := strings.ReplaceAll(line, "\x00", "")
			if strings.Contains(cleanLine, "contactPositionText") || strings.Contains(cleanLine, "emptyDash") {
				if strings.Contains(cleanLine, "func ") {
					fmt.Printf("DEFINED in %s:%d: %s\n", path, lineNum, strings.TrimSpace(cleanLine))
				} else {
					fmt.Printf("Used in %s:%d: %s\n", path, lineNum, strings.TrimSpace(cleanLine))
				}
			}
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
