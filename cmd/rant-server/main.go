package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/BadgerBadgerBadgerBadger/rant/internal/rant"
	"github.com/BadgerBadgerBadgerBadger/rant/internal/rant/ranter"
	"github.com/BadgerBadgerBadgerBadger/rant/internal/rant/slack"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	log "github.com/sirupsen/logrus"

	"github.com/BadgerBadgerBadgerBadger/goplay/pkg/config"
	"github.com/BadgerBadgerBadgerBadger/goplay/pkg/util"
)

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
	r.HandleFunc("/rant", rantSlackHandler(rantService))

	r.HandleFunc("/v1/rant", rantHandler())

	r.HandleFunc("/", indexHandler(rantService))

	// serve static files
	r.PathPrefix("/").Handler(staticHandler(rantService))

	return r
}

func indexHandler(rs *rant.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rs.Index(w)
	}
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

func rantSlackHandler(rs *rant.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		util.Must(err, "failed to parse form data")

		sc := slack.SlashCommand{}

		err = schema.NewDecoder().Decode(&sc, r.PostForm)
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

type rantRequestBody struct {
	T string `json:"t"`
}

type rantResponseBody struct {
	R string `json:"r"`
}

func rantHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		contents, err := io.ReadAll(r.Body)
		if err != nil {
			util.JSONResponse(w, "bad request")
			log.Printf("bad request, failed to read body\n")
			return
		}
		defer r.Body.Close()

		body := rantRequestBody{}
		err = json.Unmarshal(contents, &body)
		if err != nil {
			util.JSONResponse(w, "bad json")
			log.Printf("bad request, failed to decode json\n")
			return
		}

		resp := rantResponseBody{
			R: ranter.Rant(body.T),
		}

		util.JSONResponse(w, resp)
	}
}
