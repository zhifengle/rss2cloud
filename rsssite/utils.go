package rsssite

import (
	"bufio"
	"os"
	"strings"
)

func GetMagnetsFromText(textFile string) ([]string, error) {
	file, err := os.Open(textFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "magnet:?xt=urn:btih:") {
			lines = append(lines, text)
		}
	}
	return lines, scanner.Err()
}
