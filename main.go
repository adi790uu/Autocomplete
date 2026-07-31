package main

import (
	"autocomplete/internal/cache"
	"autocomplete/internal/pht"
	"encoding/json"
	"log"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/gofiber/fiber/v2"
)

type response struct {
	Query       string           `json:"query"`
	Suggestions []pht.Suggestion `json:"suggestions"`
}

func main() {
	mc := memcache.New(cache.Addr)

	app := fiber.New()

	app.Get("/suggest", func(c *fiber.Ctx) error {
		q := c.Query("q")
		if q == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing query param q"})
		}

		item, err := mc.Get(cache.Key(q))
		if err != nil {
			return c.JSON(response{Query: q, Suggestions: []pht.Suggestion{}})
		}

		var s []pht.Suggestion
		if err := json.Unmarshal(item.Value, &s); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "corrupt cache entry"})
		}

		return c.JSON(response{Query: q, Suggestions: s})
	})

	log.Fatal(app.Listen(":3000"))
}
