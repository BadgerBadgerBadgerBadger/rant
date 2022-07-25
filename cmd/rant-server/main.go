package main

import (
	"errors"
	"flag"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/BadgerBadgerBadgerBadger/rant/internal/rant"
	"github.com/BadgerBadgerBadgerBadger/rant/internal/rant/slack"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	log "github.com/sirupsen/logrus"

	"github.com/BadgerBadgerBadgerBadger/goplay/pkg/config"
	"github.com/BadgerBadgerBadgerBadger/goplay/pkg/util"
)

var decoder = schema.NewDecoder()
var indexHtml []byte

var conf rant.Config

func main() {

	rand.Seed(time.Now().UnixNano())

	configPath := flag.String("config-path", "", "provide path to the json config file")
	flag.Parse()

	if *configPath == "" {
		util.Must(errors.New("must provide a config path"))
	}
	if err := config.FromJsonFile(*configPath, &conf); err != nil {
		util.Must(err, "failed to load config")
	}

	rantService, err := rant.NewService(conf)
	util.Must(err, "failed to init rant service")

	r := initRouter(rantService)

	port := ":8000"

	log.Printf("server starting on port %s\n", port)

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(port, r))
}

func initRouter(rantService *rant.Service) *mux.Router {

	r := mux.NewRouter()

	r.Use(func(next http.Handler) http.Handler {
		return handlers.CombinedLoggingHandler(os.Stdout, next)
	})

	r.HandleFunc("/oauth", oauthHandler(rantService))

	// Routes consist of a path and a handler function.
	r.HandleFunc("/rant", rantHandler(rantService))

	// serve static files
	r.PathPrefix("/").Handler(staticHandler(rantService))

	return r
}

func staticHandler(rs *rant.Service) http.Handler {
	return rs.StaticHandler()
}

func oauthHandler(rs *rant.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		query := r.URL.Query()

		codeQ, ok := query["code"]
		if !ok || codeQ[0] == "" {
			http.Error(w, "no code available", http.StatusBadRequest)
			return
		}

		redirectUrl, err := rs.AuthSlack(codeQ[0])
		if err != nil {
			log.Warn(err.Error())
			w.WriteHeader(500)
			return
		}

		http.Redirect(w, r, redirectUrl, 301)
	}
}

func rantHandler(rs *rant.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		util.Must(err, "failed to parse form data")

		sc := slack.SlashCommand{}

		err = decoder.Decode(&sc, r.PostForm)
		if err != nil {
			http.Error(w, "Form could not be decoded", http.StatusBadRequest)
			log.WithError(err).Warn("Form could not be decoded")
			return
		}

		// we'll send a response via the response url
		err = rs.Rant(sc)
		if err != nil {
			http.Error(w, "oops, couldn't process that", http.StatusBadRequest)
			log.WithError(err).Warn("oops, couldn't process that")
			return
		}
	}
}
