package handler

// SPDX-License-Identifier: BSD-2-Clause
//
// Copyright (c) Lewis Cook <hi@lcook.net>

import (
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"

	"github.com/lcook/hottake/internal/cache"
	"github.com/lcook/hottake/internal/config"
	"github.com/lcook/hottake/internal/modal"
	"github.com/lcook/hottake/internal/version"
)

type Handler struct {
	Settings    config.Settings
	Suggestions *cache.RingBuffer[modal.Suggestion]
	Events      []any
}

func New(settings config.Settings, buffer uint64) *Handler {
	h := &Handler{
		Settings:    settings,
		Suggestions: cache.NewRingBuffer[modal.Suggestion](buffer),
	}

	h.Events = append(h.Events, h.InteractionCreate)
	h.Events = append(h.Events, h.MessageReactionAdd)

	return h
}

func (h *Handler) AggregateSuggestions(
	s *discordgo.Session,
) *discordgo.MessageEmbed {
	log.Trace("Aggregating suggestions for the last 24 hours")

	messages, _ := s.ChannelMessages(
		h.Settings.SuggestionChannel,
		100,
		"",
		"",
		"",
	)

	log.WithField("channel_id", h.Settings.SuggestionChannel).
		WithField("count", len(messages)).
		Trace("Fetched recent messages from suggestion channel")

	location, err := time.LoadLocation(h.Settings.Timezone)
	if err != nil {
		log.WithField("timezone", h.Settings.Timezone).
			Warn("Could not load timezone, falling back to UTC")

		location, _ = time.LoadLocation("UTC")
	}

	var (
		cutoff = time.Now().In(location).Add(-24 * time.Hour)
		recent = make([]string, 0, len(messages))
	)

	for _, m := range messages {
		if m.Timestamp.After(cutoff) {
			recent = append(recent, m.ID)
		}
	}

	log.WithField("count", len(recent)).Trace("Filtered recent message IDs")

	suggestions := make(map[string][]modal.Suggestion)

	for _, suggestion := range h.Suggestions.Slice() {
		if !slices.Contains(recent, suggestion.Message.ID) {
			continue
		}

		suggestions[suggestion.Platform.Name] = append(
			suggestions[suggestion.Platform.Name],
			suggestion,
		)
	}

	log.WithField("total_suggestions", len(h.Suggestions.Slice())).
		Trace("Processing stored suggestions")

	var (
		total  int
		fields = make([]*discordgo.MessageEmbedField, 0, len(suggestions))
	)

	for _, key := range slices.Collect(maps.Keys(suggestions)) {
		log.WithField("platform", key).
			WithField("count", len(suggestions[key])).
			Trace("Building weighted suggestions for platform")

		wsugs := make([]modal.WeightedSuggestion, 0, len(suggestions[key]))

		for _, suggestion := range suggestions[key] {
			message, _ := s.ChannelMessage(
				h.Settings.SuggestionChannel,
				suggestion.Message.ID,
			)

			var upvotes, downvotes int

			for _, reaction := range message.Reactions {
				switch reaction.Emoji.Name {
				case h.Settings.UpvoteEmoji:
					upvotes = reaction.Count - 1
				case h.Settings.DownvoteEmoji:
					downvotes = reaction.Count - 1
				}
			}

			wsugs = append(wsugs, modal.WeightedSuggestion{
				Suggestion: suggestion,
				Upvotes:    upvotes,
				Downvotes:  downvotes,
			})
		}

		slices.SortFunc(wsugs, func(a, b modal.WeightedSuggestion) int {
			return (b.Upvotes - (2 * b.Downvotes)) - (a.Upvotes - (2 * a.Downvotes))
		})

		var str strings.Builder

		for _, wsug := range wsugs {
			str.WriteString("- ")

			if wsug.Upvotes > 0 {
				fmt.Fprintf(
					&str,
					"%s %d ",
					h.Settings.UpvoteEmoji,
					wsug.Upvotes,
				)
			}

			if wsug.Downvotes > 0 {
				fmt.Fprintf(
					&str,
					"%s %d ",
					h.Settings.DownvoteEmoji,
					wsug.Downvotes,
				)
			}

			if wsug.Upvotes > 0 || wsug.Downvotes > 0 {
				str.WriteString(" | ")
			}

			ref := fmt.Sprintf(
				"%schannels/%s/%s/%s",
				discordgo.EndpointDiscord,
				wsug.Message.GuildID,
				wsug.Message.ChannelID,
				wsug.Message.ID,
			)

			fmt.Fprintf(
				&str,
				"[%s](<%s>) | %s",
				wsug.Title,
				wsug.URL,
				ref,
			)

			if key == "Other" {
				if link, err := url.Parse(wsug.URL); err == nil {
					fmt.Fprintf(
						&str,
						" (via %s)",
						strings.TrimPrefix(link.Hostname(), "www."),
					)
				}
			}

			if wsug.Summary != "" {
				str.WriteString(":\n > ")
				str.WriteString(wsug.Summary)
			}

			str.WriteRune('\n')

			total += 1
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   key,
			Value:  str.String(),
			Inline: false,
		})
	}

	description := "It's quiet... a little too quiet. No suggestions yet 👀"
	if total >= 1 {
		description = fmt.Sprintf(
			"Community's picks for today with %d submission(s)",
			total,
		)
	}

	return &discordgo.MessageEmbed{
		Title:       "✨ Viewer suggestions",
		Description: description,
		Color:       0x7289DA,
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf(
				"ver. %s | Got a suggestion? Use the `/suggest` command to propose content ideas!",
				version.Build,
			),
		},
	}
}
