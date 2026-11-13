package bot

import (
	"slices"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) UserOf(i *discordgo.InteractionCreate) *discordgo.User {
	if i.Member != nil {
		return i.Member.User
	}
	return i.User
}

func (b *Bot) IsAdmin(i *discordgo.InteractionCreate) bool {
	u := b.UserOf(i)

	if len(b.Cfg.Current.Authentication.Admins) == 0 {
		return false
	}

	if slices.Contains(b.Cfg.Current.Authentication.Admins, u.ID) {
		return true
	}

	return false
}

// To be added
func (b *Bot) CanUse(i *discordgo.InteractionCreate) bool {
	if len(b.Cfg.Current.Authentication.Admins) == 0 {
		return false
	}

	return false
}
