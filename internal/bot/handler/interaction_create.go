package handler

// SPDX-License-Identifier: BSD-2-Clause
//
// Copyright (c) Lewis Cook <hi@lcook.net>

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"

	"github.com/lcook/hottake/internal/modal"
)

//nolint:staticcheck
var (
	ErrRestrictedCommand = errors.New(
		"You need the <@&%s> role to use this command. Please contact a moderator to request this role",
	)
	ErrWhitelistedURL = errors.New(
		"The URL '<%s>' is not allowed for the %s platform. Please double check and try again",
	)
	ErrInvalidURL = errors.New(
		"The provided URL '%s' is invalid. Please enter a valid URL and try again",
	)
)

func (h *Handler) InteractionCreate(
	s *discordgo.Session,
	i *discordgo.InteractionCreate,
) {
	if i.GuildID == "" {
		return
	}

	log.WithFields(log.Fields{
		"type":     i.Type,
		"guild_id": i.GuildID,
		"user_id":  i.Member.User.ID,
	}).Debug("Processing Discord interaction")

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		switch i.ApplicationCommandData().Name {
		case "suggestions":
			log.WithFields(log.Fields{
				"user_id":  i.Member.User.ID,
				"guild_id": i.GuildID,
			}).Info("User requested suggestions summary")

			s.InteractionRespond(
				i.Interaction,
				&discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Flags: discordgo.MessageFlagsEphemeral,
					},
				},
			)

			s.InteractionResponseEdit(
				i.Interaction,
				&discordgo.WebhookEdit{
					Embeds: &[]*discordgo.MessageEmbed{
						h.AggregateSuggestions(s),
					},
					Flags: discordgo.MessageFlagsEphemeral,
				},
			)

		case "suggest":
			log.WithFields(log.Fields{
				"user_id":  i.Member.User.ID,
				"guild_id": i.GuildID,
			}).Info("User initiated suggestion submission")

			hasPerms := slices.Contains(
				i.Member.Roles,
				h.Settings.SubmitterRole,
			) ||
				slices.ContainsFunc(
					i.Member.Roles,
					func(id string) bool { return slices.Contains(h.Settings.DefaultRoles, id) },
				)

			if !hasPerms {
				log.WithFields(log.Fields{
					"user_id":  i.Member.User.ID,
					"guild_id": i.GuildID,
				}).Warn("User lacks permission to submit suggestions")

				s.InteractionRespond(
					i.Interaction,
					&discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf(
								ErrRestrictedCommand.Error(),
								h.Settings.SubmitterRole,
							),
							Flags: discordgo.MessageFlagsEphemeral,
						},
					},
				)

				return
			}

			form := modal.BuildSuggestionForm(
				i.Member,
				[]discordgo.Label{
					modal.BuildPlatformLabel(h.Settings.Platforms),
					modal.ContentLabel,
					modal.SummaryLabel,
					modal.ThreadLabel,
				},
			)

			err := s.InteractionRespond(i.Interaction, &form)
			if err != nil {
				log.WithError(err).Error("Failed to create interaction")
				return
			}
		}

	case discordgo.InteractionModalSubmit:
		log.WithFields(log.Fields{
			"user_id":  i.Member.User.ID,
			"guild_id": i.GuildID,
		}).Info("Processing content suggestion submission modal")

		data := i.ModalSubmitData()

		var (
			platform  = data.Components[0].(*discordgo.Label).Component.(*discordgo.SelectMenu).Values[0]
			link      = data.Components[1].(*discordgo.Label).Component.(*discordgo.TextInput).Value
			summary   = data.Components[2].(*discordgo.Label).Component.(*discordgo.TextInput).Value
			thread    = data.Components[3].(*discordgo.Label).Component.(*discordgo.SelectMenu).Values[0]
			mplatform modal.Platform
		)

		log.WithFields(log.Fields{
			"user_id":  i.Member.User.ID,
			"guild_id": i.GuildID,
			"platform": platform,
			"url":      link,
			"summary":  summary,
			"thread":   thread,
		}).Trace("Parsed modal submission data")

		for _, p := range h.Settings.Platforms {
			if platform == p.Name {
				mplatform = p
			}
		}

		suggestion := modal.NewSuggestion(i.Member, link, summary, mplatform)

		if len(suggestion.Platform.Whitelist) >= 1 &&
			!suggestion.Platform.Allowed(link) {
			log.WithFields(log.Fields{
				"user_id":   i.Member.User.ID,
				"url":       link,
				"platform":  suggestion.Platform.Name,
				"whitelist": suggestion.Platform.Whitelist,
			}).Warn("Suggestion rejected: URL not in platform whitelist")

			s.InteractionRespond(
				i.Interaction,
				&discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf(
							ErrWhitelistedURL.Error(),
							link,
							suggestion.Platform.Name,
						),
						Flags: discordgo.MessageFlagsEphemeral,
					},
				},
			)

			return
		}

		if host, err := url.Parse(link); err != nil ||
			host.Hostname() == "" {
			log.WithFields(log.Fields{
				"user_id": i.Member.User.ID,
				"url":     link,
				"error":   err,
			}).Warn("Suggestion rejected: invalid or malformed URL")

			s.InteractionRespond(
				i.Interaction,
				&discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf(ErrInvalidURL.Error(), link),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				},
			)

			return
		}

		s.InteractionRespond(
			i.Interaction,
			&discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags: discordgo.MessageFlagsEphemeral,
				},
			},
		)

		msg, err := s.ChannelMessageSendComplex(
			h.Settings.SuggestionChannel,
			&discordgo.MessageSend{
				Content: suggestion.String(),
				AllowedMentions: &discordgo.MessageAllowedMentions{
					Parse: []discordgo.AllowedMentionType{},
				},
			},
		)
		if err != nil {
			log.WithError(err).Error("Failed to send suggestion embed message")
			s.InteractionResponseEdit(
				i.Interaction,
				&discordgo.WebhookEdit{
					Content: new(
						"Failed to submit suggestion. Please try again",
					),
				},
			)

			return
		}

		for _, reaction := range []string{h.Settings.UpvoteEmoji, h.Settings.DownvoteEmoji, h.Settings.DeleteEmoji} {
			s.MessageReactionAdd(h.Settings.SuggestionChannel, msg.ID, reaction)
		}

		suggestion.Message = msg
		suggestion.Message.GuildID = h.Settings.Guild

		ref := fmt.Sprintf(
			"%schannels/%s/%s/%s",
			discordgo.EndpointDiscord,
			i.GuildID,
			msg.ChannelID,
			msg.ID,
		)

		s.InteractionResponseEdit(
			i.Interaction,
			&discordgo.WebhookEdit{
				Content: new(fmt.Sprintf(
					"Your suggestion has been submitted. View it here: %s\n\n-# If submitted in error, you can remove your suggestion by clicking on the '%s' reaction.",
					ref,
					h.Settings.DeleteEmoji,
				)),
			},
		)

		var method string

		if suggestion.Platform.Fallback == "embed" {
			suggestion.Title = modal.GetTitle(
				suggestion.Platform,
				suggestion.URL,
			)

			method = "webpage"
		}

		if suggestion.Title == "" {
			for attempt := range 5 {
				m, err := s.ChannelMessage(h.Settings.SuggestionChannel, msg.ID)
				if err != nil {
					continue
				}

				switch suggestion.Platform.Name {
				case "Twitter":
					if len(m.Embeds) >= 1 && m.Embeds[0].Author != nil {
						suggestion.Title = m.Embeds[0].Author.Name
					}
				default:
					if suggestion.Title == "" && len(m.Embeds) >= 1 {
						suggestion.Title = m.Embeds[0].Title
					}
				}

				if suggestion.Title != "" {
					log.WithField("message_id", msg.ID).
						WithField("attempt", attempt+1).
						WithField("title", suggestion.Title).
						Debug("Extracted title from Discord embed")

					break
				}

				if attempt < 4 {
					time.Sleep(
						time.Duration(250*(attempt+1)) * time.Millisecond,
					)
				}
			}

			method = "embed"

			if suggestion.Title == "" {
				suggestion.Title = strings.TrimPrefix(
					strings.TrimPrefix(suggestion.URL, "https://"),
					"http://",
				)
			}
		}

		if thread != "no" {
			_, err := s.MessageThreadStartComplex(
				h.Settings.SuggestionChannel,
				msg.ID,
				&discordgo.ThreadStart{
					Name:                suggestion.Title,
					AutoArchiveDuration: 1440,
					Invitable:           true,
				})
			if err != nil {
				log.WithError(err).Error("Unable to create thread")
			}
		}

		h.Suggestions.Add(*suggestion)

		log.WithFields(log.Fields{
			"message_id": suggestion.Message.ID,
			"user_id":    suggestion.Submitter.User.ID,
			"platform":   suggestion.Platform.Name,
			"title":      suggestion.Title,
			"summary":    suggestion.Summary,
			"url":        suggestion.URL,
			"method":     method,
			"guild_id":   i.GuildID,
		}).Info("Suggestion submitted successfully")
	}
}
