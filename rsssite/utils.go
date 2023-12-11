package rsssite

import (
	"bufio"
	"os"
	"strings"
)

func HasPrefix(str string, prefixArr []string) bool {
	for _, prefix := range prefixArr {
		if strings.HasPrefix(str, prefix) {
			return true
		}
	}
	return false
}

func GetMagnetsFromText(textFile string) ([]string, error) {
	file, err := os.Open(textFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	prefixArr := []string{"magnet:", "ed2k://", "https://", "http://", "ftp://"}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		if HasPrefix(text, prefixArr) {
			lines = append(lines, text)
		}
	}
	return lines, scanner.Err()
}
