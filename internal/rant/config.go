package rant

import (
	"github.com/BadgerBadgerBadgerBadger/rant/internal/rant/rant-store"
	"github.com/BadgerBadgerBadgerBadger/rant/internal/rant/slack"
)

type Config struct {
	Slack    slack.Config      `json:"slack"`
	Host     string            `json:"host" envconfig:"HOST"`
	Database rant_store.Config `json:"database"`
}
