package pht

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const MAX_PREFIX_LENGTH int = 20

func GeneratePrefixSuggestionMap() (map[string]*SuggestionHeap, error) {
	file, err := os.Open("data/corpus.tsv")
	if err != nil {
		fmt.Printf("Error Opening the data file")
		return nil, err
	}
	defer file.Close()
	fileReader := bufio.NewScanner(file)
	if err := fileReader.Err(); err != nil {
		fmt.Printf("error reading data file: %v\n", err)
		return nil, err
	}

	m := make(map[string]*SuggestionHeap)

	for fileReader.Scan() {
		line := fileReader.Text()
		parts := strings.Split(line, "\t")

		phrase := parts[0]
		score, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		runes := []rune(phrase)

		limit := len(runes)
		if limit > MAX_PREFIX_LENGTH {
			limit = MAX_PREFIX_LENGTH
		}

		for i := 1; i <= limit; i++ {
			prefix := string(runes[:i])
			pq, ok := m[prefix]
			if !ok {
				pq = &SuggestionHeap{}
				m[prefix] = pq
			}
			pq.Offer(Suggestion{phrase, score})
		}
	}
	return m, nil
}
