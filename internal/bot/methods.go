package bot

import (
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kopytkg/golog/v2"
	"thekrew.app/seeker/internal/feed"
	"thekrew.app/seeker/internal/sql"
)

// Helpers

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
	terms, err := b.Sql.ReadBulkTerm()
	if err != nil {
		b.Logger.ErrorWithStack(err)
	}

	ret := []pullStruct{}

	for _, term := range terms {
		uri := b.assableUri(term.Key)
		b.Logger.Debugf("uri: %s", uri)
		query, err := feed.PullFeed(uri)
		if err != nil {
			return nil, err
		}
		ret = append(ret, pullStruct{
			term: term.Key,
			data: query.Data,
		})
	}
	return &ret, nil
}

// ------------ S1 - parse data to sql model
type parseStruct struct {
	term string
	data []sql.CVEs
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
			data: []sql.CVEs{},
		}
		for _, d := range p.data {
			par.data = append(par.data,
				sql.CVEs{
					Id:          d.ID,
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
			data: []sql.CVEs{},
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

	/*
		ctx := context.Background()
		var clean []feed.Data
		for _, cve := range data {
			exists, err := b.Vk.Exists(ctx, cve.ID)
			if err != nil {
				b.Logger.Errorf("%s", err)
			}
			if !exists {
				b.Logger.Debugf("NO [%s]", cve.ID)

				err = b.Vk.Set(ctx, cve.ID, cve.ToString(), b.Cfg.Current.Cache.Expiration)
				if err != nil {
					b.Logger.Errorf("%s", err)
				}
				clean = append(clean, cve)
			} else {
				b.Logger.Debugf("Exists [%s]", cve.ID)
			}
		}
	*/

	for _, d := range *filtered {
		var tags []string
		b.Logger.Debugf("Seacher {%s}", d.term)
		u, _ := b.Sql.ReadUsersByTerm(d.term)
		if u != nil {
			for _, user := range u.Users {
				tags = append(tags, user.Tag)
				b.Logger.Debugf("tag {%s}", user.Tag)
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
