package bot

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"thekrew.app/seeker/internal/config"
	"thekrew.app/seeker/internal/sql"
)

func hexToDecimal(hex string) uint64 {
	u, err := strconv.ParseUint(hex, 16, 64)
	if err != nil {
		fmt.Print(fmt.Errorf("%w", err))
		os.Exit(1)
	}
	return u
}

func Truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	r := []rune(s)
	if len(r) > max {
		return string(r[:max-1]) + "…"
	}
	return s
}

func ToEmbed(cve sql.CVEs, cfg *config.Config) *discordgo.MessageEmbed {
	colors := map[string]int{
		"CRITICAL": int(hexToDecimal(cfg.Severities.Critical)),
		"HIGH":     int(hexToDecimal(cfg.Severities.High)),
		"MEDIUM":   int(hexToDecimal(cfg.Severities.Medium)),
		"LOW":      int(hexToDecimal(cfg.Severities.Low)),
		"NONE":     int(hexToDecimal(cfg.Severities.None)),
	}
	score := fmt.Sprintf("%.2f", cve.Score)
	if score == "0.00" && cve.Severity == "" {
		score = "NaN"
		cve.Severity = "CRITICAL"
	}

	return &discordgo.MessageEmbed{
		Title:       Truncate(cve.Id, 256),
		URL:         cve.Link,
		Description: Truncate(cve.Description, 4096),
		Color:       colors[cve.Severity],
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Score", Value: score, Inline: true},
			{Name: "Severity", Value: cve.Severity, Inline: true},
			{Name: "Vendors", Value: cve.Vendors, Inline: true},
			{Name: "Vector", Value: cve.Vector, Inline: false},
			{Name: "Published", Value: cve.CreatedAt.String(), Inline: true},
			{Name: "Updated", Value: cve.UpdatedAt.String(), Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{Text: "Seeker • CVE feed"},
	}
}

func IdToTag(id string) string {
	return fmt.Sprintf("<@%s>", id)
}
