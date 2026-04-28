package modal

// SPDX-License-Identifier: BSD-2-Clause
//
// Copyright (c) Lewis Cook <hi@lcook.net>

import (
	"github.com/bwmarrin/discordgo"
)

var ThreadLabel = discordgo.Label{
	Label:       "Discussion thread",
	Description: "Thread will be created after submission for further discussion",
	Component: discordgo.SelectMenu{
		MenuType:    discordgo.StringSelectMenu,
		CustomID:    "suggestion_thread",
		Placeholder: "Select an option...",
		Options: []discordgo.SelectMenuOption{
			{Label: "Yes", Value: "yes"},
			{Label: "No", Value: "no", Default: true},
		},
	},
}

var ContentLabel = discordgo.Label{
	Label:       "Content URL",
	Description: "Enter direct link to the content",
	Component: discordgo.TextInput{
		CustomID:  "suggestion_url",
		Style:     discordgo.TextInputShort,
		Required:  new(true),
		MaxLength: 150,
		MinLength: 15,
	},
}

var SummaryLabel = discordgo.Label{
	Label:       "Summary",
	Description: "Provide context or relevant details (optional)",
	Component: discordgo.TextInput{
		CustomID:  "suggestion_summary",
		Style:     discordgo.TextInputParagraph,
		Required:  new(false),
		MaxLength: 400,
		MinLength: 10,
	},
}

func BuildPlatformLabel(platforms []Platform) discordgo.Label {
	options := make([]discordgo.SelectMenuOption, 0, len(platforms))

	for _, platform := range platforms {
		var option discordgo.SelectMenuOption
		if platform.Default {
			option.Default = true
		}

		option.Label = platform.Name
		option.Value = platform.Name
		options = append(options, option)
	}

	return discordgo.Label{
		Label:       "Platform",
		Description: "Select the content platform",
		Component: discordgo.SelectMenu{
			MenuType:    discordgo.StringSelectMenu,
			CustomID:    "suggestion_platform",
			Placeholder: "Select platform...",
			Options:     options,
		},
	}
}

func BuildSuggestionForm(
	member *discordgo.Member,
	labels []discordgo.Label,
) discordgo.InteractionResponse {
	components := make([]discordgo.MessageComponent, 0, len(labels))

	for _, label := range labels {
		components = append(components, label)
	}

	return discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			Title:      "Content suggestion form",
			CustomID:   "suggestion_" + member.User.ID,
			Flags:      discordgo.MessageFlagsIsComponentsV2,
			Components: components,
		},
	}
}
