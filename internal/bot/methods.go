package bot

import (
	"context"
	"os"
	"slices"

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

// Ready
func (b *Bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	b.Logger.Successf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
}

func (b *Bot) SearchCVEs() {
	// assemble pull urls
	var uris []string
	terms, _ := b.Sql.ReadBulkTerm()

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
