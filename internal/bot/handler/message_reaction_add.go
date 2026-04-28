package handler

// SPDX-License-Identifier: BSD-2-Clause
//
// Copyright (c) Lewis Cook <hi@lcook.net>

import (
	"regexp"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

func ExtractUserMentions(content string) []string {
	re := regexp.MustCompile(`<@!?(\d+)>`)

	matches := re.FindAllStringSubmatch(content, -1)
	ids := make([]string, len(matches))

	for i, match := range matches {
		ids[i] = match[1]
	}

	return ids
}

func (h *Handler) MessageReactionAdd(
	s *discordgo.Session,
	m *discordgo.MessageReactionAdd,
) {
	if m.Member.User.Bot || m.GuildID == "" {
		return
	}

	msg, _ := s.ChannelMessage(m.ChannelID, m.MessageID)

	id := ExtractUserMentions(msg.Content)
	if len(id) == 0 {
		return
	}

	if m.UserID != id[0] {
		return
	}

	var emoji string
	if m.Emoji.ID != "" {
		emoji = m.Emoji.Name + ":" + m.Emoji.ID
	} else {
		emoji = m.Emoji.Name
	}

	if emoji == h.Settings.DeleteEmoji &&
		m.ChannelID == h.Settings.SuggestionChannel {
		log.WithFields(log.Fields{
			"user_id":    m.UserID,
			"message_id": m.MessageID,
			"guild_id":   m.GuildID,
		}).Info("Suggestion deleted by author")

		s.ChannelMessageDelete(m.ChannelID, m.MessageID)

		return
	}

	log.WithFields(log.Fields{
		"user_id":    m.UserID,
		"message_id": m.MessageID,
		"emoji":      emoji,
		"guild_id":   m.GuildID,
	}).Trace("Removing vote reaction from message")

	err := s.MessageReactionRemove(m.ChannelID, m.MessageID, emoji, id[0])
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"user_id":    m.UserID,
			"message_id": m.MessageID,
			"emoji":      emoji,
		}).Error("Unable to remove reaction from message")
	}
}
