package config

// SPDX-License-Identifier: BSD-2-Clause
//
// Copyright (c) Lewis Cook <hi@lcook.net>

import "github.com/lcook/hottake/internal/modal"

type BotSettings struct {
	Token       string `yaml:"token"`
	Application string `yaml:"application_id"`
	Guild       string `yaml:"guild_id"`

	DefaultRoles      []string `yaml:"default_roles"`
	SubmitterRole     string   `yaml:"submitter_role_id"`
	SuggestionChannel string   `yaml:"suggestion_channel_id"`

	UpvoteEmoji   string `yaml:"upvote_emoji"`
	DownvoteEmoji string `yaml:"downvote_emoji"`
	DeleteEmoji   string `yaml:"delete_emoji"`

	Timezone string `yaml:"timezone"`
}

type ModalSettings struct {
	Platforms []modal.Platform `yaml:"platforms"`
}
