package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/kopytkg/golog/v2"
	cron "github.com/robfig/cron/v3"
	"thekrew.app/seeker/internal/cache/valkey"
	"thekrew.app/seeker/internal/config"
	"thekrew.app/seeker/internal/feed"
)

const (
	CONFIG_SRC = "config.toml"
	VERSION    = "0.1.0"
)

func main() {
	golog.LOGLEVEL = golog.DEBUG
	logger := golog.Named("Seeker")
	logger.WithFields(golog.Fields{"version": VERSION}).Info("RSS running")

	args := os.Args
	for i, arg := range args {
		logger.Debugf("(%d) %s", i, arg)
	}
	logger.Debug("Post arg check")

	conf, err := config.LoadFromFile(CONFIG_SRC)
	if err != nil {
		if err = config.SaveDefault(CONFIG_SRC); err != nil {
			logger.Errorf("%s", err)
		}
		logger.Successf("Confing created at %s", CONFIG_SRC)
		os.Exit(1)
	}

	vk, err := valkey.New(conf.ValkeyIP)
	if err != nil {
		logger.Errorf("%f\n", err)
		os.Exit(1)
	}
	defer vk.Close()

	cron := cron.New()

	session, err := discordgo.New("Bot " + conf.Token)
	if err != nil {
		logger.Errorf("error creating Discord session, %s", err)
		return
	}

	_, err = cron.AddFunc("*/2 * * * *", func() {
		query, err := feed.PullFeed(conf.BaseUri + "?term=Microsoft&limit=50")
		if err != nil {
			golog.Error(err)
			os.Exit(1)
		}

		ctx := context.Background()
		var clean []feed.Data
		for _, cve := range query.Data {
			exists, err := vk.Exists(ctx, cve.ID)
			if err != nil {
				logger.Errorf("%s", err)
			}
			if !exists {
				logger.Debugf("NO [%s]", cve.ID)

				err = vk.Set(ctx, cve.ID, cve.ToString())
				if err != nil {
					logger.Errorf("%s", err)
				}
				clean = append(clean, cve)
			}
		}

		for _, cve := range query.Data {
			session.ChannelMessageSendEmbed("1568317626235621566", toEmbed(cve, conf))
		}

	})

	// Register the messageCreate func as a callback for MessageCreate events.
	session.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	// Open a websocket connection to Discord and begin listening.
	err = session.Open()
	if err != nil {
		logger.Errorf("error opening connection, %s", err)
		return
	}
	cron.Start()

	// Wait here until CTRL-C or other term signal is received.
	logger.Debug("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	cron.Stop()
	session.Close()

}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	golog.Debug(m.Content)
	golog.Debug(getTag(s))

	splices := strings.Split(m.Content, " ")
	if len(splices) > 1 {
		if splices[0] == getTag(s) {
			commands(splices[1:], m, s)
		}
	}
}

func getTag(s *discordgo.Session) string {
	return fmt.Sprintf("<@%s>", s.State.User.ID)
}

func createTag(id string) string {
	return fmt.Sprintf("<@%s>", id)
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

func toEmbed(cve feed.Data, cfg *config.Config) *discordgo.MessageEmbed {
	colors := map[string]int{
		"CRITICAL": int(hexToDecimal(cfg.Severities.Critical)),
		"HIGH":     int(hexToDecimal(cfg.Severities.High)),
		"MEDIUM":   int(hexToDecimal(cfg.Severities.Medium)),
		"LOW":      int(hexToDecimal(cfg.Severities.Low)),
		"NONE":     int(hexToDecimal(cfg.Severities.None)),
	}
	return &discordgo.MessageEmbed{
		Title:       truncate(cve.ID, 256),
		URL:         cve.Link,
		Description: truncate(cve.Description, 4096),
		Color:       colors[cve.Severity],
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Score", Value: fmt.Sprintf("%f", cve.Score), Inline: true},
			{Name: "Severity", Value: cve.Severity, Inline: true},
			{Name: "Vendors", Value: cve.VendorsToString(), Inline: true},
			{Name: "Vector", Value: cve.Vector, Inline: false},
			{Name: "Published", Value: feed.DateToCET(cve.CreatedAt), Inline: true},
			{Name: "Updated", Value: feed.DateToCET(cve.UpdatedAt), Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "Seeker • CVE feed"},
	}
}

func commands(args []string, m *discordgo.MessageCreate, s *discordgo.Session) {
	switch args[0] {
	case "echo":
		golog.Debugf("author ping at [%s]", createTag(m.Author.ID))
		s.ChannelMessageSend(m.ChannelID, createTag(m.Author.ID)+" Ok")

	default:
		s.ChannelMessageSend(m.ChannelID, createTag(m.Author.ID))
		golog.Debugf("[%s] not found", args[0])
	}
}
