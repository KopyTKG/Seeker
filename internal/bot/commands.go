package bot

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Slash handle
type handler func(s *discordgo.Session, i *discordgo.InteractionCreate)

var commands = []*discordgo.ApplicationCommand{
	{Name: "ping", Description: "Pong"},
	{Name: "list", Description: "List all filters"},
	{Name: "add",
		Description: "Add keyword/s into search pool",
		Options: []*discordgo.ApplicationCommandOption{{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "keyword",
			Description: "Supported formats [\"item\", \"item1,item2\"]",
			Required:    true,
		}},
	},
	{Name: "remove",
		Description: "Remove keyword/s from search pool",
		Options: []*discordgo.ApplicationCommandOption{{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "keyword",
			Description: "Supported formats [\"item\", \"item1,item2\", \"*\"]",
			Required:    true,
		}},
	},
	{Name: "follow",
		Description: "Follow topic/s in CVE feed and get notified.",
		Options: []*discordgo.ApplicationCommandOption{{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "topic",
			Description: "Supported formats [\"item\", \"item1,item2\"]",
			Required:    true,
		}},
	},
	{Name: "unfollow",
		Description: "Remove follow from topic/s",
		Options: []*discordgo.ApplicationCommandOption{{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "topic",
			Description: "Supported formats [\"item\", \"item1,item2\"]",
			Required:    true,
		}},
	},
}

// Reply
func (b *Bot) onReply(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: msg},
	})
	if err != nil {
		b.Logger.Errorf("respond: %v", err)
	}
}

// Ping
func (b *Bot) onPing(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.onReply(s, i, "pong")
}

// List
func (b *Bot) onList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	terms, _ := b.Sql.ReadBulkTerm()
	str := ""
	if len(terms) == 0 {
		str = "No search terms configured"
	} else {
		for _, t := range terms {
			str = str + " 1. " + t.Key + "\n"
		}
	}
	b.onReply(s, i, str)
}

// Add
func (b *Bot) onAdd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	term := i.ApplicationCommandData().Options[0].StringValue()
	res := "Adding \n"
	for t := range strings.SplitSeq(term, ",") {
		b.Logger.Debugf("adding item -> %s", t)
		b.Sql.CreateTerm(t)
		res = res + fmt.Sprintf("- **%s** \n", t)
	}
	res = res + "into search terms"
	b.onReply(s, i, res)
}

// Remove
func (b *Bot) onRemove(s *discordgo.Session, i *discordgo.InteractionCreate) {
	term := i.ApplicationCommandData().Options[0].StringValue()
	res := "Removing \n"
	for t := range strings.SplitSeq(term, ",") {
		if t == "*" {
			terms, _ := b.Sql.ReadBulkTerm()
			for _, tr := range terms {
				b.Logger.Debugf("removing item -> %s", tr.Key)
				b.Sql.DeleteTerm(tr.Key)
				res = res + fmt.Sprintf("- **%s** \n", tr.Key)
			}
			break
		} else {
			b.Logger.Debugf("item -> %s", t)
			b.Sql.DeleteTerm(t)
			res = res + fmt.Sprintf("- **%s** \n", t)
		}
	}

	res = res + "from search terms"
	b.onReply(s, i, res)
}

// Follow
func (b *Bot) onFollow(s *discordgo.Session, i *discordgo.InteractionCreate) {
	u := UserOf(i)
	if u == nil {
		b.onReply(s, i, "could not identify user")
		return
	}

	err := b.Sql.CreateUser(u.ID)
	if err != nil {
		b.onReply(s, i, "Database has gone rogue and decided to throw error at you.")
		b.Logger.Error(err.Error())
		return
	}

	user, err := b.Sql.ReadUser(u.ID)
	if err != nil {
		b.onReply(s, i, "Database has gone rogue and decided to throw error at you.")
		b.Logger.Error(err.Error())
		return
	}

	if user != nil {
		b.onReply(s, i, "Yaaay you now exists in DB")
	}
}

// Unfollow
func (b *Bot) onUnfollow(s *discordgo.Session, i *discordgo.InteractionCreate) {
	u := UserOf(i)
	if u == nil {
		b.onReply(s, i, "could not identify user")
		return
	}

	user, err := b.Sql.ReadUser(u.ID)
	if err != nil {
		b.onReply(s, i, IdToTag(u.ID)+" WHO THE F*CK ARE YOU??? Sorry i just never saw you.")
		b.Logger.Error(err.Error())
		return
	}

	if user != nil {

	}
}
