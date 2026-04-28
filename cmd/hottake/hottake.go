package main

// SPDX-License-Identifier: BSD-2-Clause
//
// Copyright (c) Lewis Cook <hi@lcook.net>

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	nested "github.com/antonfisher/nested-logrus-formatter"
	log "github.com/sirupsen/logrus"

	"github.com/bwmarrin/discordgo"
	"github.com/lcook/hottake/internal/bot"
	"github.com/lcook/hottake/internal/bot/handler"
)

func main() {
	var (
		cfgFile   string
		verbosity int
	)

	flag.IntVar(&verbosity, "V", 1, "Log verbosity level (1-3)")
	flag.StringVar(&cfgFile, "c", "config.yaml", "YAML configuration file path")
	flag.Parse()

	log.SetFormatter(&nested.Formatter{
		ShowFullLevel:   true,
		TrimMessages:    true,
		TimestampFormat: "[02/Jan/2006:15:04:05]",
	})

	if verbosity < 1 {
		verbosity = 1
	}

	if verbosity > 3 {
		verbosity = 3
	}

	switch verbosity {
	case 1:
		log.SetLevel(log.InfoLevel)
	case 2:
		log.SetLevel(log.DebugLevel)
	case 3:
		log.SetLevel(log.TraceLevel)
	}

reload:
	bot, err := bot.New(cfgFile)

	if err != nil {
		log.WithError(err).Fatal("Failed to initialize bot, cannot continue")
	}

	h := handler.New(bot.Settings, 100)

	commands, err := bot.Init(
		discordgo.IntentsAll,
		[]*discordgo.ApplicationCommand{
			{
				Name:        "suggest",
				Description: "Submit a new content idea suggestion",
			},
			{
				Name:        "suggestions",
				Description: "Display today's content idea suggestions",
			},
		},
		h.Events,
	)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize bot commands")
	}
	defer bot.Session.Close()

	log.Info("Bot components initialized successfully")

	sc := make(chan os.Signal, 1)
	signal.Notify(
		sc,
		os.Interrupt,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGUSR2,
	)

	switch <-sc {
	case syscall.SIGUSR2:
		log.Warn("Reload signal received, restarting bot")
		goto reload
	case os.Interrupt, syscall.SIGINT, syscall.SIGTERM:
		log.Warn("Shutdown signal received, cleaning up")
	}

	for _, command := range commands {
		err := bot.Session.ApplicationCommandDelete(
			bot.Session.State.User.ID,
			bot.Settings.Guild,
			command.ID,
		)
		if err != nil {
			log.WithError(err).
				WithField("command", command.Name).
				Warn("Failed to delete command during cleanup")
		}
	}

	log.Info("Bot gracefully stopped")
}
