package modal

// SPDX-License-Identifier: BSD-2-Clause
//
// Copyright (c) Lewis Cook <hi@lcook.net>

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"

	"github.com/bwmarrin/discordgo"
)

type Suggestion struct {
	Message             *discordgo.Message
	Submitter           *discordgo.Member
	URL, Summary, Title string
	Platform            Platform
}

type WeightedSuggestion struct {
	Suggestion
	Upvotes, Downvotes int
}

func NewSuggestion(
	submitter *discordgo.Member,
	url, summary string,
	platform Platform,
) *Suggestion {
	p := &Suggestion{
		Submitter: submitter,
		URL:       url,
		Summary:   summary,
		Platform:  platform,
	}

	return p
}

func (s *Suggestion) String() string {
	var str strings.Builder
	fmt.Fprintf(&str, "[%s](%s) ", s.Platform.Name, s.URL)

	if s.Platform.Name == "Other" {
		content, _ := url.Parse(s.URL)
		fmt.Fprintf(
			&str,
			"(via %s) ",
			strings.TrimPrefix(content.Hostname(), "www."),
		)
	}

	fmt.Fprintf(&str, "• Suggestion posted by %s", s.Submitter.Mention())

	if s.Summary != "" {
		str.WriteString(":\n> " + s.Summary)
	}

	return str.String()
}

func GetTitle(platform Platform, url string) string {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	switch platform.Name {
	case "Reddit":
		resp, err := client.Get(url)
		if err != nil {
			return ""
		}

		resp.Body.Close()

		resp, err = client.Get(
			strings.Split(resp.Request.URL.String(), "?")[0] + ".json",
		)
		if err != nil {
			return ""
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.WithFields(log.Fields{
				"url":         url,
				"status_code": resp.StatusCode,
			}).Error("Error fetching webpage")

			return ""
		}

		var reddit []struct {
			Data struct {
				Children []struct {
					Data struct {
						Title     string `json:"title"`
						Subreddit string `json:"subreddit"`
					} `json:"data"`
				} `json:"children"`
			} `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&reddit); err != nil {
			return ""
		}

		if len(reddit) > 0 && len(reddit[0].Data.Children) > 0 {
			title := fmt.Sprintf(
				"%s (r/%s)",
				reddit[0].Data.Children[0].Data.Title,
				reddit[0].Data.Children[0].Data.Subreddit,
			)

			log.WithField("url", url).
				WithField("title", title).
				Trace("Fetched title from Reddit post")

			return title
		}

		return ""
	default:
		resp, err := client.Get(url)
		if err != nil {
			return ""
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.WithFields(log.Fields{
				"url":         url,
				"status_code": resp.StatusCode,
			}).Error("Error fetching page")

			return ""
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"url": url,
			}).Error("Unable to parse HTML webpage")

			return ""
		}

		if ogTitle, exists := doc.Find("meta[property='og:title']").
			Attr("content"); exists && ogTitle != "" {
			log.WithField("url", url).
				WithField("title", ogTitle).
				Trace("Fetched title from meta tag")

			return ogTitle
		}

		if title := doc.Find("title").Text(); title != "" {
			trimmed := strings.TrimSpace(title)
			log.WithField("url", url).
				WithField("title", trimmed).
				Trace("Fetched title from HTML title tag")

			return trimmed
		}

		return ""
	}
}
