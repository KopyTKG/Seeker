package main

import (
	"context"
	"os"

	"github.com/kopytkg/golog/v2"
	"thekrew.app/cve-discord/internal/cache/valkey"
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

	vk, err := valkey.New(conf.ValkeyIP)
	if err != nil {
		logger.Errorf("%f\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	var clean []rss.CVE
	for _, cve := range CVEs {
		exists, err := vk.Exists(ctx, cve.NO)
		if err != nil {
			golog.Errorf("%s", err)
		}
		if !exists {
			logger.Debugf("NO [%s] | Title [%s]", cve.NO, cve.Title)
			err := vk.Set(ctx, cve.NO, cve.ToString())
			if err != nil {
				golog.Errorf("%s", err)
			}
			clean = append(clean, cve)
		}
	}

	err = webhook.SendEmbed(conf, clean)
	if err != nil {
		logger.Errorf("Err: %s", err)
		os.Exit(1)
	}
}
