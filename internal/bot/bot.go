package bot

import (
	"flag"
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
	Vk      *valkey.Client
	Cfg     *config.Manager
	Sql     *sqlite.SQLite
	Logger  *golog.Logger
	Session *discordgo.Session
	Cron    *cron.Cron
}

func NewBot(path string, version string) *Bot {
	logger := golog.Named("Seeker")

	/*
		=== conf.Current.g file handeling ===
	*/
	conf, err := config.LoadFromFile(path)
	if err != nil {
		if err = config.SaveDefault(path); err != nil {
			logger.Errorf("%s", err)
		}
		logger.Successf("conf.Current.ng created at %s", path)
		os.Exit(1)
	}

	switch conf.Current.Logging.Loglevel {
	case "DEBUG":
		golog.LOGLEVEL = golog.DEBUG
	case "ERROR":
		golog.LOGLEVEL = golog.ERROR
	case "WARN":
		golog.LOGLEVEL = golog.WARN
	case "INFO":
		golog.LOGLEVEL = golog.INFO
	default:
		golog.LOGLEVEL = golog.ERROR
	}

	if conf.Current.Logging.Logfile != "" {
		golog.EnableLogFile(conf.Current.Logging.Logfile)
	}

	if conf.Current.OwnerID == "" {
		logger.Errorf("Bot cannot run without owner!! Set OwnerID in {%s}", path)
		os.Exit(9)
	}

	/*
		=== Valkey cache ===
	*/
	vk, err := valkey.New(conf.Current.ValkeyIP)
	if err != nil {
		logger.Errorf("%f\n", err)
		os.Exit(1)
	}

	/*
		=== SQLite memory ===
	*/
	c, err := sqlite.NewSQL(conf.Current.Database)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	/*
		=== Bot setup ===
	*/
	session, err := discordgo.New("Bot " + conf.Current.Token)
	if err != nil {
		logger.Errorf("error creating Discord session, %s", err)
		os.Exit(1)
	}

	// In this example, we only care about receiving message events.
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent

	b := Bot{
		Vk:      vk,
		Cfg:     conf,
		Sql:     c,
		Logger:  logger,
		Session: session,
		Cron:    cron.New(),
	}

	b.AddHandler(b.onReady)

	var handlers = map[string]handler{
		"ping":     b.onPing,
		"list":     b.onList,
		"set":      b.onSet,
		"add":      b.onAdd,
		"remove":   b.onRemove,
		"follow":   b.onFollow,
		"unfollow": b.onUnfollow,
		"follows":  b.onFollows,
	}

	b.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}
		if h, ok := handlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	_, err = b.AddFunc("*/2 * * * *", b.SearchCVEs)
	if err != nil {
		b.Logger.Errorf("Error during cron creation, %s", err)
		os.Exit(1)
	}

	return &b
}

// Wrapper
func (b *Bot) AddHandler(fn any) {
	b.Session.AddHandler(fn)
}

// Cron wrapper
func (b *Bot) AddFunc(interval string, fn func()) (cron.EntryID, error) {
	return b.Cron.AddFunc(interval, fn)
}

func (b *Bot) Run() {
	defer b.Vk.Close()
	defer b.Sql.Close()
	defer b.Cron.Stop()
	defer b.Session.Close()

	// Open a websocket connection to Discord and begin listening.
	err := b.Session.Open()
	if err != nil {
		b.Logger.Errorf("error opening connection, %s", err)
		return
	}
	b.Cron.Start()

	guild := flag.String("guild", b.Cfg.Current.Guild, "guild ID, empty = global")

	if _, err := b.Session.ApplicationCommandBulkOverwrite(b.Session.State.User.ID, *guild, commands); err != nil {
		b.Logger.ErrorWithStack(err)
		os.Exit(1)
	}

	// Wait here until CTRL-C or other term signal is received.
	b.Logger.Debug("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
