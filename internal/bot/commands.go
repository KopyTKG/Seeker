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

func (b *Bot) onSet(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.IsAdmin(i) {
		b.onReply(s, i, "Not Ok")
	} else {
		b.Logger.Debugf("setting channel [%s] as default", i.ChannelID)
		b.Cfg.Current.Authentication.ChannelID = i.ChannelID
		b.Cfg.Save()
		b.onReply(s, i, "Setting current channel as feed channel")
	}
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

func getArg(i *discordgo.InteractionCreate) string {
	return strings.ToLower(i.ApplicationCommandData().Options[0].StringValue())
}

// Add
func (b *Bot) onAdd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	term := getArg(i)
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
	term := getArg(i)
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
			b.Sql.DeleteTerm(strings.ToLower(t))
			res = res + fmt.Sprintf("- **%s** \n", t)
		}
	}

	res = res + "from search terms"
	b.onReply(s, i, res)
}

// Follow
func (b *Bot) onFollow(s *discordgo.Session, i *discordgo.InteractionCreate) {
	term := getArg(i)
	u := b.UserOf(i)
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

	res := "You are now following \n"
	if user != nil {
		for t := range strings.SplitSeq(term, ",") {
			if t == "*" {
				terms, _ := b.Sql.ReadBulkTerm()
				for _, tr := range terms {
					b.Logger.Debugf("item -> %s", tr.Key)
					b.Sql.CreateUserTerm(tr.Id, u.ID)
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
				b.Sql.CreateUserTerm(st.Id, u.ID)
				res = res + "- **" + st.Key + "** \n"

			}
		}

	}
	res = res + "You will be pinged if search includes your topic"
	b.onReply(s, i, res)
}

// Follows
func (b *Bot) onFollows(s *discordgo.Session, i *discordgo.InteractionCreate) {
	u := b.UserOf(i)
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
		info, err := b.Sql.ReadTermsByUser(u.ID)
		if err != nil {
			b.onReply(s, i, IdToTag(u.ID)+" You are'nt following any topic.")
			b.Logger.Error(err.Error())
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

}

// Unfollow
func (b *Bot) onUnfollow(s *discordgo.Session, i *discordgo.InteractionCreate) {
	term := getArg(i)
	u := b.UserOf(i)
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

	res := "You are no longer following these topics: \n"
	if user != nil {
		for t := range strings.SplitSeq(term, ",") {
			if t == "*" {
				terms, _ := b.Sql.ReadBulkTerm()
				for _, tr := range terms {
					b.Logger.Debugf("item -> %s", tr.Key)
					err := b.Sql.DeleteUserTerm(tr.Key, u.ID)
					if err != nil {
						b.Logger.Error(err.Error())
					}
					res = res + "- **" + tr.Key + "** \n"
				}
				break
			} else {
				b.Logger.Debugf("item -> %s", t)
				err := b.Sql.DeleteUserTerm(t, u.ID)
				if err != nil {
					b.Logger.Error(err.Error())
				}
				res = res + "- **" + t + "** \n"

			}
		}

	}
	b.onReply(s, i, res)

}
