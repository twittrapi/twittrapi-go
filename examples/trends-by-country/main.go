// Command trends-by-country prints the current trends for a country.
//
//	TWITTRAPI_API_KEY=... go run ./examples/trends-by-country DE
//
// Argument: two-letter country code (default US), or "world" for worldwide trends.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	twittrapi "github.com/twittrapi/twittrapi-go"
	"github.com/twittrapi/twittrapi-go/examples/internal/extract"
)

func main() {
	code := "US"
	if len(os.Args) > 1 {
		code = strings.ToUpper(strings.TrimSpace(os.Args[1]))
	}

	client, err := twittrapi.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	var woeid int64 = twittrapi.WOEIDWorldwide
	label := "Worldwide"
	if code != "WORLD" {
		// Expected shape: [{"name", "woeid", "countryCode", "placeType": {"name": "Country"}}, ...]
		raw, err := client.Trends.Locations(ctx)
		if err != nil {
			log.Fatal(err)
		}
		data, err := extract.Decode(raw)
		if err != nil {
			log.Fatal(err)
		}
		found := false
		extract.Walk(data, func(m map[string]any) {
			if found || m["countryCode"] != code {
				return
			}
			placeType := m["placeType"]
			if pt, ok := placeType.(map[string]any); ok {
				placeType = pt["name"]
			}
			id, ok := m["woeid"].(float64)
			if placeType != "Country" || !ok {
				return
			}
			woeid = int64(id)
			label, _ = m["name"].(string)
			found = true
		})
		if !found {
			fmt.Fprintf(os.Stderr, "No trend location found for country code %s\n", code)
			os.Exit(1)
		}
	}

	raw, err := client.Trends.Get(ctx, woeid)
	if err != nil {
		log.Fatal(err)
	}
	data, err := extract.Decode(raw)
	if err != nil {
		log.Fatal(err)
	}

	// Collect every object that looks like a trend ({"name": ...}) inside a "trends" array.
	type trend struct {
		name   string
		volume float64
	}
	var trends []trend
	var find func(v any)
	find = func(v any) {
		switch t := v.(type) {
		case []any:
			for _, x := range t {
				find(x)
			}
		case map[string]any:
			if list, ok := t["trends"].([]any); ok {
				for _, item := range list {
					if m, ok := item.(map[string]any); ok {
						if name, ok := m["name"].(string); ok {
							vol, _ := m["tweet_volume"].(float64)
							trends = append(trends, trend{name, vol})
						}
					}
				}
			}
			for _, x := range t {
				find(x)
			}
		}
	}
	find(data)

	fmt.Printf("Trends for %s (WOEID %d):\n", label, woeid)
	if len(trends) == 0 {
		pretty, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(pretty))
	}
	for i, t := range trends {
		volume := ""
		if t.volume > 0 {
			volume = fmt.Sprintf(" (%.0f posts)", t.volume)
		}
		fmt.Printf("%2d. %s%s\n", i+1, t.name, volume)
	}
}
