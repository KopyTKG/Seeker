package main

import (
	"os"

	disbot "thekrew.app/seeker/internal/bot"
)

const (
	CONFIG_SRC = "config.toml"
	VERSION    = "0.1.0"
)

func main() {

	bot := disbot.NewBot(CONFIG_SRC, VERSION)

	args := os.Args
	for i, arg := range args {
		bot.Logger.Debugf("(%d) %s", i, arg)
	}
	bot.Logger.Debug("Post arg check")

	_, err := bot.AddFunc("*/2 * * * *", bot.SearchCVEs)
	if err != nil {
		bot.Logger.Errorf("Error during cron creation, %s", err)
		return
	}

	bot.Run()
}
