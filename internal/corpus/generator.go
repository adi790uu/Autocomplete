package corpus

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
)

const targetCount = 500000

func Generator() {
	file, err := os.Open("/usr/share/dict/words")
	if err != nil {
		fmt.Printf("error opening dictionary: %v\n", err)
		return
	}
	defer file.Close()

	var words []string
	reader := bufio.NewScanner(file)
	for reader.Scan() {
		word := strings.ToLower(reader.Text())
		words = append(words, word)
	}
	if err := reader.Err(); err != nil {
		fmt.Printf("error reading dictionary: %v\n", err)
		return
	}
	fmt.Printf("Length of Words: %d\n", len(words))

	rng := rand.New(rand.NewSource(42))

	phrases := make([]string, 0, targetCount)
	seen := make(map[string]bool)

	for len(phrases) < targetCount {
		length := pickLength(rng)

		parts := make([]string, length)
		for i := range length {
			parts[i] = words[rng.Intn(len(words))]
		}
		phrase := strings.Join(parts, " ")

		if seen[phrase] {
			continue
		}
		seen[phrase] = true
		phrases = append(phrases, phrase)
	}

	if err := os.MkdirAll("data", 0o755); err != nil {
		fmt.Printf("error creating data dir: %v\n", err)
		return
	}
	out, err := os.Create("data/corpus.tsv")
	if err != nil {
		fmt.Printf("error creating corpus file: %v\n", err)
		return
	}
	defer out.Close()

	w := bufio.NewWriter(out)
	defer w.Flush()

	for i, phrase := range phrases {
		rank := i + 1
		score := targetCount / rank
		fmt.Fprintf(w, "%s\t%d\n", phrase, score)
	}

	fmt.Printf("wrote %d phrases to data/corpus.tsv\n", len(phrases))
}

func pickLength(rng *rand.Rand) int {
	roll := rng.Intn(100)
	switch {
	case roll < 50:
		return 1
	case roll < 80:
		return 2
	case roll < 95:
		return 3
	default:
		return 4
	}
}
