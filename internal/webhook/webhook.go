package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"thekrew.app/cve-discord/internal/config"
	"thekrew.app/cve-discord/internal/rss"
)

type Payload struct {
	Username  string  `json:"username,omitempty"`
	AvatarURL string  `json:"avatar_url,omitempty"`
	Content   string  `json:"content,omitempty"`
	Embeds    []Embed `json:"embeds"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type Thumbnail struct {
	Url string `json:"url"`
}

type EmbedFooter struct {
	Text string `json:"text"`
}

type Embed struct {
	Title       string       `json:"title,omitempty"`
	Url         string       `json:"url,omitempty"`
	Author      *EmbedAuthor `json:"author,omitempty"`
	Description string       `json:"description,omitempty"`
	Color       int          `json:"color,omitempty"`
	Fields      []EmbedField `json:"fields,omitempty"`
	Footer      *EmbedFooter `json:"footer,omitempty"`
	Timestamp   string       `json:"timestamp,omitempty"`
}

type EmbedAuthor struct {
	Name    string `json:"name"`
	IconURL string `json:"icon_url,omitempty"`
}

// WebhookRequest represents a single webhook payload.
type WebhookRequest struct {
	Link string
	Data []byte
}

func hexToDecimal(hex string) uint64 {
	u, err := strconv.ParseUint(hex, 16, 64)
	if err != nil {
		fmt.Print(fmt.Errorf("%w", err))
		os.Exit(1)
	}
	return u
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	r := []rune(s)
	if len(r) > max {
		return string(r[:max-1]) + "…"
	}
	return s
}

// ExecuteWebhook sends a webhook request with rate-limit handling.
func ExecuteWebhook(link string, data []byte) error {
	// Create the HTTP request.
	req, err := http.NewRequest("POST", link, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read the response body.
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode == 200 || resp.StatusCode == 204 {
		return nil
	}

	if resp.StatusCode == 429 {
		return nil
	}

	// Handle unexpected status codes.
	return fmt.Errorf("Unexpected status code %d: %s", resp.StatusCode, bodyText)
}

// SendEmbed sends an embed to a Discord webhook.
func SendEmbed(cfg *config.Config, cve []rss.CVE) error {

	colors := map[string]uint64{
		"CRITICAL": hexToDecimal(cfg.ColorConfig.CriticalColor),
		"HIGH":     hexToDecimal(cfg.ColorConfig.HighColor),
	}

	for _, c := range cve {
		payload := Payload{
			Username:  cfg.UserConfig.Username,
			AvatarURL: cfg.UserConfig.AvatarURL,
			Embeds: []Embed{{
				Title:     truncate(c.Title, 256),
				Url:       c.Link,
				Color:     (int)(colors[c.Color]),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Author:    &EmbedAuthor{Name: truncate(c.Severity, 256), IconURL: cfg.UserConfig.AvatarURL},
				Fields: []EmbedField{{
					Name:   c.NO,
					Value:  truncate(c.Description, 1024),
					Inline: true,
				}, {
					Name:   "Published At",
					Value:  c.PublishedAt,
					Inline: false,
				}},
			}},
		}

		p, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		err = ExecuteWebhook(cfg.Webhook, p)
		if err != nil {
			return err
		}
		time.Sleep(500 * time.Millisecond)

	}
	return nil

}
