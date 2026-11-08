package bot

import (
	"context"
	"os"
	"slices"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/kopytkg/golog/v2"
	"thekrew.app/seeker/internal/feed"
	"thekrew.app/seeker/internal/sql"
)

// Helpers
func (b *Bot) assableUri(terms string) string {
	return b.Cfg.BaseUri + "?term=" + terms
}
func stringTerms(terms []sql.Term) string {
	s := ""
	for i, t := range terms {
		if i < len(terms)-1 {
			s = s + t.Key + ","
		} else {
			s = s + t.Key
		}
	}
	return s
}

func (b *Bot) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.ChannelID != b.Cfg.ChannelID {
		return
	}

	b.Logger.Debugf("%s", m.Content)
	err := b.Sql.AddUser(IdToTag(m.Author.ID))
	if err != nil {
		b.Logger.ErrorWithStack(err)
		return
	}

	splices := strings.Split(m.Content, " ")
	if len(splices) > 1 {
		if splices[0] == b.getTag(s) {
			switch splices[1] {
			case "setFilter":
				if len(splices) < 3 {
					b.Session.ChannelMessageSend(m.ChannelID, IdToTag(m.Author.ID)+" No `term` givven")
					break
				}
				b.addFilter(m, splices[2:])
			case "listFilters":
				b.listFilters(m)
			default:
				b.helpMenu(m)
			}

		}
	} else {
		b.helpMenu(m)
	}
}

func (b *Bot) helpMenu(m *discordgo.MessageCreate) {
	// To be added

	b.Session.ChannelMessageSend(m.ChannelID, IdToTag(m.Author.ID)+`
- echo
- setFilter {item}
- listFilters
		`)
}

func (b *Bot) addFilter(m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		b.Session.ChannelMessageSend(m.ChannelID, IdToTag(m.Author.ID)+" No `term` givven")
	}

	for _, t := range args {
		b.Logger.Debugf("item -> %s", t)
		b.Sql.AddTerm(t)
	}
}

func (b *Bot) listFilters(m *discordgo.MessageCreate) {
	terms, _ := b.Sql.GetTerms()
	b.Session.ChannelMessageSend(m.ChannelID, IdToTag(m.Author.ID)+" "+stringTerms(terms))
}

func (b *Bot) SearchCVEs() {
	// assemble pull urls
	var uris []string
	terms, _ := b.Sql.GetTerms()

	if len(terms) > 10 {
		for chunk := range slices.Chunk(terms, 10) {
			uris = append(uris, b.assableUri(stringTerms(chunk)))
		}
	} else {
		uris = append(uris, b.assableUri(stringTerms(terms)))
	}

	var data []feed.Data

	for _, uri := range uris {
		b.Logger.Debugf("uri: %s", uri)
		query, err := feed.PullFeed(uri)
		if err != nil {
			golog.Error(err)
			os.Exit(1)
		}
		data = append(data, query.Data...)
	}

	ctx := context.Background()
	var clean []feed.Data
	for _, cve := range data {
		exists, err := b.Vk.Exists(ctx, cve.ID)
		if err != nil {
			b.Logger.Errorf("%s", err)
		}
		if !exists {
			b.Logger.Debugf("NO [%s]", cve.ID)

			err = b.Vk.Set(ctx, cve.ID, cve.ToString())
			if err != nil {
				b.Logger.Errorf("%s", err)
			}
			clean = append(clean, cve)
		}
	}

	for _, cve := range data {
		b.Session.ChannelMessageSendEmbed("1568317626235621566", ToEmbed(cve, b.Cfg))
	}
}
