package main

import (
	"fmt"
	"os"

	"github.com/kopytkg/golog/v2"
	disbot "thekrew.app/seeker/internal/bot"
)

const (
	CONFIG_SRC = "config.toml"
	VERSION    = "0.1.0"
)

func main() {
	golog.LOGLEVEL = golog.DEBUG
	arg, err := getArgs(os.Args)

	if err != nil {
		fmt.Print(err.Error())
		os.Exit(2)
	}

	if arg == "" {
		arg = CONFIG_SRC
	}

	bot := disbot.NewBot(arg, VERSION)
	_, err = bot.AddFunc("*/2 * * * *", bot.SearchCVEs)
	if err != nil {
		bot.Logger.Errorf("Error during cron creation, %s", err)
		return
	}

	bot.Run()
}

func getArgs(args []string) (string, error) {
	help, cfg := false, false

	path := ""

	for _, arg := range args {
		switch arg {
		case "--help":
			help = true
		case "-h":
			help = true
		case "-c":
			cfg = true
		case "--conf":
			cfg = true
		default:
			if cfg {
				path = arg
				cfg = false
			}

		}

		if help {
			PrintHelp()
			os.Exit(0)
		}

	}
	if path != "" {
		return path, nil
	}
	return "", nil
}

func PrintHelp() {
	helpText := `Seeker
Usage: seeker [?args]

ARGUMENTS:
  -h, --help                Display help menu
  -c, --conf {file}	    Use specific config path

Report issues to: <https://github.com/kopytkg/seeker/issues>
`
	fmt.Print(helpText)
}
