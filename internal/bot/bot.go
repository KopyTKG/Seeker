package bot

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/kopytkg/golog/v2"
	"github.com/robfig/cron/v3"
	"thekrew.app/seeker/internal/cache/valkey"
	"thekrew.app/seeker/internal/config"
	"thekrew.app/seeker/internal/sql/sqlite"
)

type Bot struct {
	vk      *valkey.Client
	cfg     *config.Config
	sql     *sqlite.SQLite
	logger  *golog.Logger
	session *discordgo.Session
	cron    *cron.Cron
}

func NewBot(path string, version string) *Bot {
	golog.LOGLEVEL = golog.DEBUG
	logger := golog.Named("Seeker")
	logger.WithFields(golog.Fields{"version": version}).Info("RSS running")

	/*
		=== Config file handeling ===
	*/
	conf, err := config.LoadFromFile(path)
	if err != nil {
		if err = config.SaveDefault(path); err != nil {
			logger.Errorf("%s", err)
		}
		logger.Successf("Confing created at %s", path)
		os.Exit(1)
	}

	/*
		=== Valkey cache ===
	*/
	vk, err := valkey.New(conf.ValkeyIP)
	if err != nil {
		logger.Errorf("%f\n", err)
		os.Exit(1)
	}
	defer vk.Close()

	/*
		=== SQLite memory ===
	*/
	c, err := sqlite.NewSQL("test.db")
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer c.Close()

	/*
		=== Bot setup ===
	*/
	session, err := discordgo.New("Bot " + conf.Token)
	if err != nil {
		logger.Errorf("error creating Discord session, %s", err)
		os.Exit(1)
	}

	// In this example, we only care about receiving message events.
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	return &Bot{
		vk:      vk,
		cfg:     conf,
		sql:     c,
		logger:  logger,
		session: session,
		cron:    cron.New(),
	}
}

// Wrapper
func (b *Bot) AddHandler(fn func()) {
	b.session.AddHandler(fn)
}

// Cron wrapper
func (b *Bot) AddFunc(interval string, fn func()) (cron.EntryID, error) {
	return b.cron.AddFunc(interval, fn)
}

func (b *Bot) Run() {

	// Open a websocket connection to Discord and begin listening.
	err := b.session.Open()
	if err != nil {
		b.logger.Errorf("error opening connection, %s", err)
		return
	}
	b.cron.Start()

	// Wait here until CTRL-C or other term signal is received.
	b.logger.Debug("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	b.cron.Stop()
	b.session.Close()
}
