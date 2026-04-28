package bot

// SPDX-License-Identifier: BSD-2-Clause
//
// Copyright (c) Lewis Cook <hi@lcook.net>

import (
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"

	"github.com/lcook/hottake/internal/config"
	"github.com/lcook/hottake/internal/version"
)

type Bot struct {
	Settings config.Settings
	Session  *discordgo.Session
}

func New(path string) (*Bot, error) {
	log.WithField("file", path).Debug("Loading configuration from file")

	settings, err := config.FromFile[config.Settings](path)
	if err != nil {
		log.WithError(err).
			WithField("file", path).
			Error("Unable to load configuration file")
		return nil, err
	}

	log.Debug("Configuration loaded successfully")

	log.Debug("Initializing Discord session")

	session, err := discordgo.New("Bot " + settings.Token)
	if err != nil {
		log.WithError(err).Error("Failed to create Discord session")
		return nil, err
	}

	log.Debug("Discord session created")

	return &Bot{
		Settings: settings,
		Session:  session,
	}, nil
}

func (b *Bot) Run(
	intents discordgo.Intent,
	commands []*discordgo.ApplicationCommand,
	handlers ...[]any,
) ([]*discordgo.ApplicationCommand, error) {
	b.Session.Identify.Intents = intents
	b.Session.State.MaxMessageCount = 500

	log.Debug("Establishing connection to Discord")

	err := b.Session.Open()
	if err != nil {
		log.WithError(err).Error("Unable to connect to Discord")
		return nil, err
	}

	log.Info("Connected to Discord successfully")

	var _handlers []any

	for _, slice := range handlers {
		_handlers = append(_handlers, slice...)
	}

	if len(_handlers) > 0 {
		log.WithField("count", len(_handlers)).
			Debug("Registering Discord event handlers")

		for _, handler := range _handlers {
			b.Session.AddHandler(handler)
		}
	}

	var _commands []*discordgo.ApplicationCommand

	if len(commands) > 0 {
		log.WithField("count", len(commands)).
			Debug("Registering Discord application commands")

		for _, command := range commands {
			cmd, err := b.Session.ApplicationCommandCreate(
				b.Settings.Application,
				b.Settings.Guild,
				command,
			)
			if err != nil {
				log.WithError(err).
					WithField("command", command.Name).
					Error("Failed to register Discord command")

				continue
			}

			_commands = append(_commands, cmd)
		}

		log.WithField("count", len(_commands)).
			Info("Discord commands registered successfully")
	}

	return _commands, nil
}

func (b *Bot) Init(
	intents discordgo.Intent,
	commands []*discordgo.ApplicationCommand,
	handlers ...[]any,
) ([]*discordgo.ApplicationCommand, error) {
	commands, err := b.Run(intents, commands, handlers...)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"bot_id":   b.Session.State.User.ID,
		"username": b.Session.State.User.Username,
		"guild_id": b.Settings.Guild,
		"commands": len(commands),
		"version":  version.Build,
	}).Info("Bot initialized and ready")

	return commands, nil
}
