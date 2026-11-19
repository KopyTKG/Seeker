package feed

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"unicode"

	"github.com/kopytkg/golog/v2"
)

func capitalize(str string) string {
	if len(str) == 0 {
		return ""
	}

	runes := []rune(str)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func DateToCET(date string, timezone string) string {
	tz, err := time.LoadLocation(timezone)
	if err != nil {
		golog.Errorf("Error during timezone setup: %s", err)
	}

	layout := "2006-01-02T15:04:05Z07:00"
	time, err := time.Parse(layout, date)
	if err != nil {
		golog.Errorf("Error during datetime parsing: %s", err)
		return ""
	}

	return time.In(tz).Local().String()
}

func PullFeed(uri string) (*Query, error) {
	if uri == "" {
		return nil, fmt.Errorf("URL is empty %s", "")
	}

	res, err := http.Get(uri)

	if err != nil {
		return nil, fmt.Errorf("Error during api fetch: %s", err)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("Error during body fetching: %s", err)
	}

	var j Query

	if err = json.Unmarshal(data, &j); err != nil {
		return nil, fmt.Errorf("Error during body parsing: %s", err)
	}

	return &j, nil
}
