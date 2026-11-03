package rss

import (
	"context"
	"fmt"
	"strings"

	"github.com/ungerik/go-rss"
)

type CVE struct {
	Link        string
	NO          string
	Title       string
	Description string
	Severity    string
	PublishedAt string
	Color       string
}

func (cv *CVE) ToString() string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s", cv.Link, cv.NO, cv.Title, cv.Description, cv.Severity, cv.PublishedAt, cv.Color)
}

func stringLength(str string) int {
	var length int
	for range str {
		length++
	}
	return length
}

func PullRSS(url string) ([]CVE, error) {
	ctx := context.Background()
	resp, err := rss.Read(ctx, url, false)
	if err != nil {
		return nil, fmt.Errorf("failed to download RSS Feed: %s", err)
	}
	defer resp.Body.Close()

	feed, err := rss.Regular(ctx, resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSS Feed: %s", err)
	}

	var CVEs []CVE

	for _, entry := range feed.Item {

		lines := strings.Split(entry.Description, "\n")
		var clean []string
		for _, line := range lines {
			if stringLength(line) > 0 && stringLength(line) != 8 {
				if strings.Contains(line, "</strong>") {
					clean = append(clean, strings.Split(line, "</strong>")[1])
				}
			}
		}
		i := 1
		cve := CVE{Link: entry.Link, Title: strings.Split(entry.Title, " - ")[1]}
		for _, line := range clean {
			switch i {
			case 1:
				cve.NO = line
			case 2:
				cve.PublishedAt = line
			case 3:
				cve.Description = line
			case 4:
				cve.Severity = line
				cve.Color = strings.Split(line, " | ")[1]
			}
			i += 1

			if i > 4 {
				i = 1
				CVEs = append(CVEs, cve)
				cve = CVE{Link: entry.Link}
			}
		}

	}

	return CVEs, nil

}
