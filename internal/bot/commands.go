package bot

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Slash handle
type handler func(s *discordgo.Session, i *discordgo.InteractionCreate)

var commands = []*discordgo.ApplicationCommand{
	{Name: "ping", Description: "Pong"},
	{Name: "list", Description: "List all filters"},
	{Name: "add",
		Description: "Add keyword to be search in CVE feed",
		Options: []*discordgo.ApplicationCommandOption{{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "term",
			Description: "\"item\" or \"item1,item2\" are only supported",
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
	terms, _ := b.Sql.GetTerms()
	b.onReply(s, i, stringTerms(terms))
}

// Add
func (b *Bot) onAdd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	term := i.ApplicationCommandData().Options[0].StringValue()
	for _, t := range strings.Split(term, ",") {
		b.Logger.Debugf("item -> %s", t)
		b.Sql.AddTerm(t)
	}

	b.onReply(s, i, "Added {"+term+"} in search terms")
}
