package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"

	discordwebhook "github.com/bensch777/discord-webhook-golang"
	"github.com/kopytkg/golog/v2"
	"github.com/ungerik/go-rss"
	"thekrew.app/cve-discord/internal/config"
)

type Config struct {
	Webhook  string `toml:"Webhook"`
	RSSFeed  string `toml:"RSSFeed"`
	ATOMFeed string `toml:"ATOMFeed"`
}

const (
	CONFIG_SRC = "config.toml"
	VERSION    = "0.0.1"
)

type Payload struct {
	Username  string  `json:"username,omitempty"`
	AvatarURL string  `json:"avatar_url,omitempty"`
	Content   string  `json:"content,omitempty"`
	Embeds    []Embed `json:"embeds"`
}

type EmbedAuthor struct {
	Name    string `json:"name"`
	IconURL string `json:"icon_url,omitempty"`
}

type Embed struct {
	Title       string       `json:"title,omitempty"`
	URL         string       `json:"url,omitempty"`
	Author      *EmbedAuthor `json:"author,omitempty"`
	Description string       `json:"description,omitempty"`
	Color       int          `json:"color,omitempty"`
	Fields      []EmbedField `json:"fields,omitempty"`
	Footer      *EmbedFooter `json:"footer,omitempty"`
	Timestamp   string       `json:"timestamp,omitempty"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type EmbedFooter struct {
	Text string `json:"text"`
}

type EmbedPayload struct {
	Embeds []Embed `json:"embeds"`
}

type CVE struct {
	Link        string
	NO          string
	Title       string
	Description string
	Severity    string
	PublishedAt string
	Color       string
}

func stringLength(str string) int {
	var length int
	for range str {
		length++
	}
	return length
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

func main() {
	colors := map[string]int{
		"CRITICAL": 16389140,
		"HIGH":     16097063,
	}

	golog.LOGLEVEL = golog.DEBUG
	logger := golog.Named("RSS")
	logger.WithFields(golog.Fields{"version": VERSION}).Info("RSS running")

	conf, err := config.LoadFromFile(CONFIG_SRC)
	if err != nil {
		golog.Error(err)
		os.Exit(1)
	}

	ctx := context.Background()
	resp, err := rss.Read(ctx, conf.RSSFeed, false)
	if err != nil {
		logger.Errorf("failed to download RSS Feed: %s", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	feed, err := rss.Regular(ctx, resp)
	if err != nil {
		logger.Errorf("failed to parse RSS Feed: %s", err)
		os.Exit(1)
	}

	var CVEs []CVE

	for _, entry := range feed.Item {

		lines := strings.Split(entry.Description, "\n")
		var clean []string
		for _, line := range lines {
			if stringLength(line) > 0 && stringLength(line) != 8 {
				if strings.Contains(line, "</strong>") {
					clean = append(clean, strings.Split(line, "</strong>")[1])
				}
			}
		}
		i := 1
		cve := CVE{Link: entry.Link, Title: strings.Split(entry.Title, " - ")[1]}
		for _, line := range clean {
			switch i {
			case 1:
				cve.NO = line
				logger.Debug(line)
			case 2:
				cve.PublishedAt = line
				logger.Debug(line)
			case 3:
				cve.Description = line
				logger.Debug(line)
			case 4:
				cve.Severity = line
				cve.Color = strings.Split(line, " | ")[1]
				logger.Debug(line)
			}
			i += 1

			if i > 4 {
				i = 1
				CVEs = append(CVEs, cve)
				cve = CVE{Link: entry.Link}
			}
		}

	}

	err = sendEmbed(conf, CVEs, colors)
	if err != nil {
		logger.Errorf("Err: %s", err)
		os.Exit(1)
	}
}

func sendEmbed(cfg *config.Config, cve []CVE, colors map[string]int) error {
	logger := golog.Named("RSS")
	const icon = "https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/png/arch-linux.png"

	for _, c := range cve {
		logger.Debugf("cve: %s", c.NO)
		logger.Debugf("cve: %s", c.PublishedAt)
		logger.Debugf("cve: %s", c.Description)
		logger.Debugf("cve: %s", c.Severity)

		payload := Payload{
			Username:  "❤️❤️UwU Archie UwU ❤️❤️",
			AvatarURL: icon,
			Embeds: []Embed{{
				Title:     truncate(c.Title, 256),
				URL:       c.Link,
				Color:     colors[c.Color],
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Author:    &EmbedAuthor{Name: truncate(c.Severity, 256), IconURL: icon},
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

		body, err := json.Marshal(payload)
		if err != nil {
			logger.ErrorWithStack(err)
			os.Exit(1)
		}
		if err := discordwebhook.ExecuteWebhook(cfg.Webhook, body); err != nil {
			logger.ErrorWithStack(err)
			os.Exit(1)
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}
