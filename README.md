# autocomplete

A prefix-based autocomplete backend in Go, built to learn and benchmark the
architecture described in Walmart's engineering blog:

> **[How we rebuilt the Walmart Autocomplete Backend](https://medium.com/walmartglobaltech/how-we-rebuilt-the-walmart-autocomplete-backend-10efe71d624a)**

This project is a hands-on implementation of that blog post. The goal was to
reproduce their "prefix hash tree + distributed cache" design on a laptop and
measure how it behaves.

## What the blog describes (and what this implements)

The Walmart post replaces a runtime Ternary Search Tree with a design that does
all the expensive work **offline** and serves queries as plain cache reads:

| Idea from the blog | Where it lives here |
|---|---|
| **Prefix hash tree** — a `HashMap<Prefix, Array>` of every prefix mapped to its pre-sorted top suggestions, instead of traversing a tree at query time | `internal/pht/prefix_suggestion_map.go` |
| **Bounded priority queue** — keep only the top `S` suggestions per prefix during the offline build, avoiding `O(M log M)` sorting per query | `internal/pht/heap.go` (min-heap bounded to `S = 8`) |
| **Serialize the prefix hash tree into a distributed cache (Memcached)** as the primary serving store | `cmd/loader` writes every prefix into Memcached |
| **Read-only serving path** — the online service just does a cache lookup | `main.go` (Fiber server, `GET /suggest`) |
| **Daily cron rebuild** re-populates the cache | re-run `cmd/loader` (Memcached has no persistence) |

The main simplification vs. Walmart: their cache keys encode AB-test variant,
experience, and category (e.g. `aeE~app`); here the key is just the prefix.

## Architecture

```
                    OFFLINE (build)                         ONLINE (serve)
  ┌──────────────┐   ┌──────────────┐   ┌───────────┐        ┌──────────────┐
  │ cmd/corpus   │──▶│ cmd/loader   │──▶│ Memcached │◀──────▶│ main.go       │
  │ generate     │   │ build prefix │   │ (primary  │  GET   │ Fiber server  │
  │ corpus.tsv   │   │ hash tree,   │   │  store)   │        │ /suggest?q=   │
  └──────────────┘   │ SET all keys │   └───────────┘        └──────────────┘
                     └──────────────┘
```

The serving path holds **no in-process map** — every request is a single
Memcached `GET` + JSON decode.

## Project layout

```
main.go                              Fiber API server (read-only, serves from Memcached)
cmd/corpus/main.go                   generate data/corpus.tsv from the system dictionary
cmd/loader/main.go                   build the prefix hash tree and load it into Memcached
internal/pht/prefix_suggestion_map.go  build map: prefix -> bounded top-S heap
internal/pht/heap.go                 bounded min-heap of suggestions (Suggestion, SuggestionHeap)
internal/cache/cache.go              shared Memcached address + key encoding
internal/corpus/generator.go         synthetic corpus generator
```

## Prerequisites

- Go 1.24+
- Memcached (`brew install memcached`)
- A word list at `/usr/share/dict/words` (present by default on macOS/Linux) —
  only needed to regenerate the corpus

## Running

Order matters: **generate corpus → start Memcached → load → serve.**

```bash
# 1. Generate the corpus (500k synthetic phrases with Zipf-style scores)
#    Produces data/corpus.tsv. Skip if the file already exists.
go run ./cmd/corpus

# 2. Start Memcached with enough memory to hold every prefix (~3.6M keys).
#    64MB is NOT enough — it silently LRU-evicts. Use ~2GB.
memcached -p 11211 -m 2048 -d

# 3. Build the prefix hash tree and load it into Memcached
go run ./cmd/loader
#    -> loaded 3630828 prefixes into memcached at 127.0.0.1:11211

# 4. Run the API server
go run .
#    -> listening on :3000
```

> **Note:** Memcached is not persistent. If you restart it (or it evicts under
> memory pressure), re-run `cmd/loader`. This mirrors Walmart's daily cron
> re-populate.

## API

`GET /suggest?q=<prefix>`

```bash
curl "http://localhost:3000/suggest?q=hel"
```

```json
{
  "query": "hel",
  "suggestions": [
    { "Text": "hellebore", "Score": 2475 },
    { "Text": "helmetmaking", "Score": 175 }
  ]
}
```

Suggestions are returned pre-ranked (highest `Score` first). An unknown prefix
returns an empty list.

## Benchmarking

The serving path is pure Memcached reads, so a load test measures roughly what
the blog reports (sub-1ms cache reads, low-ms end-to-end).

```bash
brew install hey
hey -n 20000 -c 100 "http://localhost:3000/suggest?q=hel"
```

### Sample result

20,000 requests at concurrency 100, single process, over loopback on a laptop:

| Metric | Value |
|---|---|
| Throughput | **41,687 req/s** |
| Average latency | 2.3 ms |
| p50 | 2.1 ms |
| p95 | 4.8 ms |
| p99 | 9.8 ms |
| Slowest | 18 ms |
| `resp wait` (server + Memcached, avg) | 2.2 ms |
| Status codes | 20,000 × `200` (no errors) |

`resp wait` is the real backend cost: Fiber routing → Memcached `GET` → JSON
decode → response. This lines up with the blog's reported sub-1ms cache reads
and <10ms end-to-end (here p99 = 9.8ms).

**Caveats:** every request hits the same key (`q=hel`), so this is best-case
hot-key cache behavior; loopback has no real network hop; and `hey` reuses
keep-alive connections, so connection setup isn't measured. Real traffic spread
across millions of prefixes and a real network would be higher.

Keep concurrency realistic (`-c 50`–`200`). On macOS the default listen backlog
is small (`kern.ipc.somaxconn = 128`), so bursting `-c 1000` at loopback causes
`connection reset by peer` — that measures the OS accept queue, not the backend.
To push higher concurrency:

```bash
sudo sysctl -w kern.ipc.somaxconn=1024   # then restart the server
memcached -p 11211 -m 2048 -c 4096 -d    # raise Memcached's connection cap
```

## Corpus format

`data/corpus.tsv` is tab-separated `phrase<TAB>score`, where a higher score means
more popular:

```
pocketing	500000
grouped shyness	250000
nondeprivable	166666
```

The generator builds 1–4 word phrases from the system dictionary and assigns
`score = 500000 / rank`, giving a Zipf-like popularity distribution.
