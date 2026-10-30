package main

import (
	"os"

	"github.com/kopytkg/golog/v2"
	"thekrew.app/cve-discord/internal/config"
	"thekrew.app/cve-discord/internal/rss"
	"thekrew.app/cve-discord/internal/webhook"
)

const (
	CONFIG_SRC = "config.toml"
	VERSION    = "0.0.1"
)

func main() {
	golog.LOGLEVEL = golog.DEBUG
	logger := golog.Named("RSS")
	logger.WithFields(golog.Fields{"version": VERSION}).Info("RSS running")

	conf, err := config.LoadFromFile(CONFIG_SRC)
	if err != nil {
		golog.Error(err)
		os.Exit(1)
	}

	CVEs, err := rss.PullRSS(conf.RSSFeed)
	if err != nil {
		golog.Error(err)
		os.Exit(1)
	}

	err = webhook.SendEmbed(conf, CVEs)
	if err != nil {
		logger.Errorf("Err: %s", err)
		os.Exit(1)
	}
}
