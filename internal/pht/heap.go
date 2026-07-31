package pht

import (
	"container/heap"
	"sort"
)

// S is the bounded size — the max number of suggestions we keep per prefix
const S = 8

// Suggestion is one autocomplete candidate: the text and its popularity score.
// Higher Score = more popular = should rank first.
type Suggestion struct {
	Text  string
	Score int
}

type SuggestionHeap []Suggestion

var _ heap.Interface = (*SuggestionHeap)(nil)

func (h SuggestionHeap) Len() int {
	return len(h)
}

func (h SuggestionHeap) Less(i, j int) bool {
	return h[i].Score < h[j].Score
}

func (h SuggestionHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *SuggestionHeap) Push(x any) {
	*h = append(*h, x.(Suggestion))
}

func (h *SuggestionHeap) Pop() any {
	old := *h
	n := len(old)
	last := old[n-1]
	*h = old[:n-1]
	return last
}

// Ranked returns the heap's suggestions ordered best-first (highest Score).
// It copies, so it does not disturb the underlying heap.
func (h SuggestionHeap) Ranked() []Suggestion {
	out := make([]Suggestion, len(h))
	copy(out, h)
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

func (h *SuggestionHeap) Offer(s Suggestion) {
	heapSlice := *h
	length := len(heapSlice)
	if length < S {
		heap.Push(h, s)
		return
	}
	if s.Score > heapSlice[0].Score {
		heap.Pop(h)
		heap.Push(h, s)
	}
}
