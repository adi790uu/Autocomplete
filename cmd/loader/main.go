package main

import (
	"autocomplete/internal/cache"
	"autocomplete/internal/pht"
	"encoding/json"
	"log"

	"github.com/bradfitz/gomemcache/memcache"
)

func main() {
	m, err := pht.GeneratePrefixSuggestionMap()
	if err != nil {
		log.Fatalf("build prefix map: %v", err)
	}

	mc := memcache.New(cache.Addr)

	loaded := 0
	for prefix, heap := range m {
		b, err := json.Marshal(heap.Ranked())
		if err != nil {
			continue
		}
		if err := mc.Set(&memcache.Item{Key: cache.Key(prefix), Value: b}); err != nil {
			log.Fatalf("set %q: %v", prefix, err)
		}
		loaded++
	}

	log.Printf("loaded %d prefixes into memcached at %s", loaded, cache.Addr)
}
