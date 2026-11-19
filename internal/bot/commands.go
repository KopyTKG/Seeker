package bot

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	_ "thekrew.app/seeker/internal/sql"
)

// Slash handle
type handler func(s *discordgo.Session, i *discordgo.InteractionCreate)

var commands = []*discordgo.ApplicationCommand{
	{Name: "ping", Description: "Pong"},
	{Name: "list", Description: "List all filters"},
	{Name: "set", Description: "Set current channel as feed channel"},
	{Name: "addrole",
		Description: "Add role into trusted group",
		Options: []*discordgo.ApplicationCommandOption{{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "roletag",
			Description: "@Role to be set as trusted role",
			Required:    true,
		}},
	},
	{Name: "delrole",
		Description: "Delete role from trusted group",
		Options: []*discordgo.ApplicationCommandOption{{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "roletag",
			Description: "@Role to be removed from trusted role",
			Required:    true,
		}},
	},
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
	{Name: "follows", Description: "List all topic user follows"},
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

func getArg(i *discordgo.InteractionCreate) string {
	return strings.ToLower(i.ApplicationCommandData().Options[0].StringValue())
}

func (b *Bot) noPerms(s *discordgo.Session, i *discordgo.InteractionCreate) {
	b.onReply(s, i, "You have insuficient permissions to use this command")
}

// Ready
func (b *Bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	b.Logger.Successf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	err := s.UpdateStatusComplex(discordgo.UpdateStatusData{
		Status: "online",
		Activities: []*discordgo.Activity{{
			Name: "for new CVEs (" + b.Version + ")",
			Type: discordgo.ActivityTypeWatching,
		}},
	})
	if err != nil {
		b.Logger.ErrorWithStack(err)
	}
}

// Set
func (b *Bot) onSet(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.IsAdmin(i) {
		b.noPerms(s, i)
	} else {
		b.Logger.Debugf("setting channel [%s] as default", i.ChannelID)
		b.Cfg.Current.Authentication.ChannelID = i.ChannelID
		b.Cfg.Save()
		b.onReply(s, i, "Setting current channel as feed channel")
	}
}

// AddRole
func (b *Bot) onAddRole(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.IsAdmin(i) {
		b.noPerms(s, i)
	} else {
		roleTag := getArg(i)
		id := (roleTag[:len(roleTag)-1])[3:]
		b.Logger.Debugf("setting role [%s] as trusted", id)
		b.Cfg.Current.Authentication.Roles = append(b.Cfg.Current.Authentication.Roles, id)
		b.Cfg.Save()
		b.onReply(s, i, "Adding role into trusted group")
	}
}

// DelRole
func (b *Bot) onDelRole(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.IsAdmin(i) {
		b.noPerms(s, i)
	} else {
		roleTag := getArg(i)
		id := (roleTag[:len(roleTag)-1])[3:]
		b.Logger.Debugf("removing role [%s] from trusted", id)
		roles := []string{}
		for _, r := range b.Cfg.Current.Authentication.Roles {
			if r != id {
				roles = append(roles, r)
			}
		}
		b.Cfg.Current.Authentication.Roles = roles
		b.Cfg.Save()
		b.onReply(s, i, "Removing role from trusted group")
	}
}

// Reply
func (b *Bot) onReply(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlags(discordgo.MessageFlagsEphemeral),
		},
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
	if b.CanUse(i) {
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
	} else {
		b.noPerms(s, i)
	}

}

// Add
func (b *Bot) onAdd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if b.CanUse(i) {
		term := getArg(i)
		res := "Adding \n"
		for t := range strings.SplitSeq(term, ",") {
			b.Logger.Debugf("adding item -> %s", t)
			err := b.Sql.CreateTerm(t)
			if err != nil {
				b.Logger.ErrorWithStack(err)
				b.onReply(s, i, "Error has occured durring adding")
				return
			}
			res = res + fmt.Sprintf("- **%s** \n", t)
		}
		res = res + "into search terms"
		b.onReply(s, i, res)
	} else {
		b.noPerms(s, i)
	}
}

// Remove
func (b *Bot) onRemove(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if b.CanUse(i) {
		term := getArg(i)
		res := "Removing \n"
		for t := range strings.SplitSeq(term, ",") {
			if t == "*" {
				terms, _ := b.Sql.ReadBulkTerm()
				for _, tr := range terms {
					b.Logger.Debugf("removing item -> %s", tr.Key)
					err := b.Sql.DeleteTerm(tr.Key)
					if err != nil {
						b.Logger.ErrorWithStack(err)
						b.onReply(s, i, "Error has occured durring removing")
						return
					}
					res = res + fmt.Sprintf("- **%s** \n", tr.Key)
				}
				break
			} else {
				b.Logger.Debugf("item -> %s", t)
				err := b.Sql.DeleteTerm(strings.ToLower(t))
				if err != nil {
					b.Logger.ErrorWithStack(err)
					b.onReply(s, i, "Error has occured durring removing")
					return

				}
				res = res + fmt.Sprintf("- **%s** \n", t)
			}
		}

		res = res + "from search terms"
		b.onReply(s, i, res)
	} else {
		b.noPerms(s, i)
	}
}

// Follow
func (b *Bot) onFollow(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if b.CanUse(i) {
		term := getArg(i)
		u := b.UserOf(i)
		if u == nil {
			b.onReply(s, i, "could not identify user")
			return
		}

		err := b.Sql.CreateUser(u.ID)
		if err != nil {
			b.onReply(s, i, "Database has gone rogue and decided to throw error at you.")
			b.Logger.ErrorWithStack(err)
			return
		}

		user, err := b.Sql.ReadUser(u.ID)
		if err != nil {
			b.onReply(s, i, "Database has gone rogue and decided to throw error at you.")
			b.Logger.ErrorWithStack(err)
			return
		}

		res := "You are now following \n"
		if user != nil {
			for t := range strings.SplitSeq(term, ",") {
				if t == "*" {
					terms, _ := b.Sql.ReadBulkTerm()
					for _, tr := range terms {
						b.Logger.Debugf("item -> %s", tr.Key)
						err := b.Sql.CreateUserTerm(tr.Id, u.ID)
						if err != nil {
							b.onReply(s, i, "Database has gone rogue and decided to throw error at you.")
							b.Logger.ErrorWithStack(err)
							return
						}
						res = res + "- **" + tr.Key + "** \n"
					}
					break
				} else {
					b.Logger.Debugf("searching {%s}", t)
					st, err := b.Sql.ReadTerm(t)
					if err != nil {
						b.onReply(s, i, "Topic not found")
						return
					}
					b.Logger.Debugf("item -> %s", st.Key)
					err = b.Sql.CreateUserTerm(st.Id, u.ID)
					if err != nil {
						b.onReply(s, i, "Database has gone rogue and decided to throw error at you.")
						b.Logger.ErrorWithStack(err)
						return
					}
					res = res + "- **" + st.Key + "** \n"

				}
			}

		}
		res = res + "You will be pinged if search includes your topic"
		b.onReply(s, i, res)
	} else {
		b.noPerms(s, i)
	}
}

// Follows
func (b *Bot) onFollows(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if b.CanUse(i) {
		u := b.UserOf(i)
		if u == nil {
			b.onReply(s, i, "could not identify user")
			return
		}

		user, err := b.Sql.ReadUser(u.ID)
		if err != nil {
			b.onReply(s, i, IdToTag(u.ID)+" WHO THE F*CK ARE YOU??? Sorry i just never saw you.")
			b.Logger.ErrorWithStack(err)
			return
		}

		if user != nil {
			info, err := b.Sql.ReadTermsByUser(u.ID)
			if err != nil {
				b.onReply(s, i, IdToTag(u.ID)+" You are'nt following any topic.")
				b.Logger.ErrorWithStack(err)
				return
			}
			if len(info.Terms) == 0 {
				b.onReply(s, i, IdToTag(u.ID)+" You are'nt following any topic.")
				return
			}
			res := "You are following \n"
			for _, topic := range info.Terms {
				res = res + "- **" + topic.Key + "**\n"
			}
			b.onReply(s, i, res)
		} else {
			b.onReply(s, i, IdToTag(u.ID)+" How the f*ck did you get here?? Like ... you f*cked my code so badly i got here.")
		}
	} else {
		b.noPerms(s, i)
	}

}

// Unfollow
func (b *Bot) onUnfollow(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if b.CanUse(i) {
		term := getArg(i)
		u := b.UserOf(i)
		if u == nil {
			b.onReply(s, i, "could not identify user")
			return
		}

		user, err := b.Sql.ReadUser(u.ID)
		if err != nil {
			b.onReply(s, i, IdToTag(u.ID)+" WHO THE F*CK ARE YOU??? Sorry i just never saw you.")
			b.Logger.ErrorWithStack(err)
			return
		}

		res := "You are no longer following these topics: \n"
		if user != nil {
			for t := range strings.SplitSeq(term, ",") {
				if t == "*" {
					terms, _ := b.Sql.ReadBulkTerm()
					for _, tr := range terms {
						b.Logger.Debugf("item -> %s", tr.Key)
						err := b.Sql.DeleteUserTerm(tr.Key, u.ID)
						if err != nil {
							b.Logger.ErrorWithStack(err)
							return
						}
						res = res + "- **" + tr.Key + "** \n"
					}
					break
				} else {
					b.Logger.Debugf("item -> %s", t)
					err := b.Sql.DeleteUserTerm(t, u.ID)
					if err != nil {
						b.Logger.ErrorWithStack(err)
						return
					}
					res = res + "- **" + t + "** \n"

				}
			}

		}
		b.onReply(s, i, res)
	} else {
		b.noPerms(s, i)
	}

}
