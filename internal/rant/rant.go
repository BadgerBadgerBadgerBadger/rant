package rant

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/BadgerBadgerBadgerBadger/goplay/pkg/util"
	"github.com/BadgerBadgerBadgerBadger/rant/internal/rant/rant-store"
	"github.com/BadgerBadgerBadgerBadger/rant/internal/rant/ranter"
	"github.com/BadgerBadgerBadgerBadger/rant/internal/rant/slack"
	"github.com/gorilla/schema"
	"github.com/pkg/errors"
)

//go:embed static/*
var static embed.FS

var saynonyms = []string{
	"says",
	"intones",
	"announces",
	"conveys",
	"expresses",
	"speaks",
	"gabs",
	"flaps",
	"orates",
	"puts forth",
	"makes known",
	"yaks",
	"verbalizes",
	"utters",
	"opines",
	"recites",
	"remarks",
	"communicates",
}

type Service struct {
	config            Config
	staticFileHandler http.Handler
	slackClient       *slack.Client
	decoder           *schema.Decoder
}

func NewService(c Config) (*Service, error) {

	s := Service{
		config:  c,
		decoder: schema.NewDecoder(),
	}

	slackStore, err := rant_store.NewRantStore(c.Database)
	if err != nil {
		return nil, err
	}

	s.slackClient = slack.NewClient(c.Slack, slackStore)

	staticFs, err := fs.Sub(fs.FS(static), "static")
	if err != nil {
		return nil, errors.Wrap(err, "failed to static file system")
	}

	s.staticFileHandler = http.FileServer(http.FS(staticFs))

	return &s, nil
}

func (rs *Service) StaticHandler() http.Handler {
	return rs.staticFileHandler
}

func (rs *Service) AuthSlack(code string) (string, error) {

	err := rs.slackClient.Authenticate(code)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://%s/success.html", rs.config.Host), nil
}

func (rs *Service) Rant(sc slack.SlashCommand) error {

	genedRant := fmt.Sprintf(
		"<@%s> %s:\n %s",
		sc.UserId,
		util.Sample(saynonyms),
		ranter.Rant(sc.Text),
	)

	return rs.slackClient.SendRant(sc, genedRant)
}
