package bot

import (
	"context"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kopytkg/golog/v2"
	"gorm.io/gorm"
	"thekrew.app/seeker/internal/feed"
	"thekrew.app/seeker/internal/infra/model"
)

// Helpers

func stringTerms(terms []model.Term) string {
	s := ""
	for i, t := range terms {
		if i < len(terms)-1 {
			s = s + t.Query + ","
		} else {
			s = s + t.Query
		}
	}
	return s
}

/*
--------------------------------------------
Stages
--------------------------------------------
*/

// ------------ S0 - get data
type pullStruct struct {
	term string
	data []feed.Data
}

func (b *Bot) assableUri(terms string) string {
	u, err := url.Parse(b.Cfg.Current.Configuration.BaseUri)
	if err != nil {
		b.Logger.Errorf("invalid base uri: %s", err)
		return b.Cfg.Current.Configuration.BaseUri
	}
	q := u.Query()
	q.Set("term", terms)
	u.RawQuery = q.Encode()
	return u.String()
}

func (b *Bot) pullData() (*[]pullStruct, error) {
	ctx := context.Background()
	terms, err := gorm.G[model.Term](b.Database.ORM).Find(ctx)
	if err != nil {
		b.Logger.ErrorWithStack(err)
	}

	ret := []pullStruct{}

	for _, term := range terms {
		uri := b.assableUri(term.Query)
		b.Logger.Debugf("uri: %s", uri)
		query, err := feed.PullFeed(uri)
		if err != nil {
			return nil, err
		}
		ret = append(ret, pullStruct{
			term: term.Query,
			data: query.Data,
		})
	}
	return &ret, nil
}

// ------------ S1 - parse data to sql model
type parseStruct struct {
	term string
	data []model.CVE
}

func stringToDate(date string, timezone string) time.Time {
	tz, err := time.LoadLocation(timezone)
	if err != nil {
		golog.Errorf("Error during timezone setup: %s", err)
	}

	layout := "2006-01-02T15:04:05Z07:00"
	time, err := time.Parse(layout, date)
	if err != nil {
		golog.Errorf("Error during datetime parsing: %s", err)
	}

	return time.In(tz).Local()
}

func (b *Bot) parseData(pull *[]pullStruct) *[]parseStruct {
	ret := []parseStruct{}
	tz := b.Cfg.Current.Configuration.TimeZone

	for _, p := range *pull {
		par := parseStruct{
			term: p.term,
			data: []model.CVE{},
		}
		for _, d := range p.data {
			par.data = append(par.data,
				model.CVE{
					ID:          d.ID,
					Title:       d.Title,
					Description: d.Description,
					CreatedAt:   stringToDate(d.CreatedAt, tz),
					UpdatedAt:   stringToDate(d.UpdatedAt, tz),
					Score:       d.Score,
					Severity:    d.Severity,
					Vector:      d.Vector,
					Vendors:     d.VendorsToString(),
					Link:        d.Link,
				},
			)
		}
		ret = append(ret, par)
	}
	return &ret
}

// ------------ S2 - filter data to have TTL max 3 days (CVE updated in last 80h)
func (b *Bot) filterData(parsed *[]parseStruct) *[]parseStruct {
	ret := []parseStruct{}

	for _, p := range *parsed {
		par := parseStruct{
			term: p.term,
			data: []model.CVE{},
		}

		for _, d := range p.data {
			now := time.Now()
			update := d.UpdatedAt

			diff := now.Sub(update)
			if diff.Abs().Hours() < 80 {
				par.data = append(par.data, d)
			}
		}
		ret = append(ret, par)
	}

	return &ret
}

func (b *Bot) SearchCVEs() {
	pulls, err := b.pullData() // Stage 0 - get data
	if err != nil {
		b.Logger.ErrorWithStack(err)
		os.Exit(1)
	}
	parsed := b.parseData(pulls)     // Stage 1 - parse data to SQL schema format
	filtered := b.filterData(parsed) // Stage 2 - filter data so they are max 80h old

	ctx := context.Background()
	var clean []parseStruct

	for _, bundle := range *filtered {
		p := parseStruct{
			term: bundle.term,
			data: []model.CVE{},
		}
		for _, cve := range bundle.data {
			exists, err := b.Vk.Exists(ctx, cve.ID)
			if err != nil {
				b.Logger.Errorf("%s", err)
			}
			if !exists {
				b.Logger.Debugf("NO [%s]", cve.ID)

				err = b.Vk.Set(ctx, cve.ID, bundle.term, b.Cfg.Current.Cache.Expiration)
				if err != nil {
					b.Logger.Errorf("%s", err)
				}
				p.data = append(p.data, cve)
			} else {
				b.Logger.Debugf("Exists [%s]", cve.ID)
			}
		}
		clean = append(clean, p)
	}

	for _, d := range clean {
		var tags []string
		b.Logger.Debugf("Seacher {%s}", d.term)
		u, _ := gorm.G[model.User](b.Database.ORM).Preload("Terms", func(db gorm.PreloadBuilder) error {
			db.Where("Query = ?", d.term)
			return nil
		}).Find(ctx)
		if u != nil {
			for _, user := range u {
				tags = append(tags, IdToTag(user.Tag))
				b.Logger.Debugf("tag {%s}", IdToTag(user.Tag))
			}
		}

		for _, cve := range d.data {
			b.Session.ChannelMessageSendComplex(b.Cfg.Current.Authentication.ChannelID, &discordgo.MessageSend{
				Content: strings.Join(tags, ","),
				Embeds:  []*discordgo.MessageEmbed{ToEmbed(cve, b.Cfg.Current)},
				AllowedMentions: &discordgo.MessageAllowedMentions{
					Parse: []discordgo.AllowedMentionType{discordgo.AllowedMentionTypeUsers},
				},
			})
		}
	}
}
