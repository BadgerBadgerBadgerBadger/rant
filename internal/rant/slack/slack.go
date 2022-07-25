package slack

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/BadgerBadgerBadgerBadger/goplay/pkg/util"
)

type SlashCommand struct {
	Token               string `schema:"token"`
	Command             string `schema:"command"`
	Text                string `schema:"text"`
	ResponseURL         string `schema:"response_url"`
	TriggerID           string `schema:"trigger_id"`
	UserId              string `schema:"user_id"`
	UserName            string `schema:"user_name"`
	TeamID              string `schema:"team_id"`
	ChannelID           string `schema:"channel_id"`
	ChannelName         string `schema:"channel_name"`
	APIAppID            string `schema:"api_app_id"`
	IsEnterpriseInstall bool   `schema:"is_enterprise_install"`
	TeamDomain          string `schema:"team_domain"`
}

type ResponseTpe string

type CommandReply struct {
	ResponseType    ResponseTpe `json:"response_type"`
	Text            string      `json:"text"`
	ReplaceOriginal bool        `json:"replace_original"`
	DeleteOriginal  bool        `json:"delete_original"`
}

const (
	ephemeral ResponseTpe = "ephemeral"
	inChannel ResponseTpe = "in_channel"
)

type Config struct {
	Oauth OauthConfig `json:"oauth"`
}

type OauthConfig struct {
	ClientID     string `json:"client_id" envconfig:"SLACK_OAUTH_CLIENT_ID"`
	ClientSecret string `json:"client_secret" envconfig:"SLACK_OAUTH_CLIENT_SECRET"`
	RedirectUrl  string `json:"redirect_url" envconfig:"SLACK_OAUTH_REDIRECT_URL"`
}

type Client struct {
	config Config
	store  Store
}

func NewClient(config Config, store Store) *Client {
	return &Client{
		config: config,
		store:  store,
	}
}

type OauthResponse struct {
	Ok         bool       `json:"ok"`
	Error      *string    `json:"error"`
	AuthedUser AuthedUser `json:"authed_user"`
}

func (s *Client) Authenticate(code string) error {
	authReq, err := http.NewRequest("GET", "https://slack.com/api/oauth.v2.access", nil)
	util.Must(err, "failed to create new request")

	authReqQuery := authReq.URL.Query()
	authReqQuery.Set("code", code)
	authReqQuery.Set(
		"redirect_uri",
		s.config.Oauth.RedirectUrl,
	)
	authReq.URL.RawQuery = authReqQuery.Encode()

	basicAuthTokenRaw := fmt.Sprintf("%s:%s", s.config.Oauth.ClientID, s.config.Oauth.ClientSecret)
	basicAuthToken := base64.StdEncoding.EncodeToString([]byte(basicAuthTokenRaw))

	authReq.Header.Set("Authorization", fmt.Sprintf("Basic %s", basicAuthToken))

	res, err := (&http.Client{}).Do(authReq)
	defer res.Body.Close()

	if err != nil {
		return errors.Wrap(err, "failed to call slack oauth")
	}

	if res.StatusCode != http.StatusOK {
		return errors.New("oauth responded with non-ok status")
	}

	oauthResp := OauthResponse{}
	err = json.NewDecoder(res.Body).
		Decode(&oauthResp)
	if err != nil {
		return errors.Wrapf(err, "failed to read oauth response body")
	}

	if !oauthResp.Ok {
		return errors.New(*oauthResp.Error)
	}

	err = s.store.StoreAuthedUser(oauthResp.AuthedUser.ID, oauthResp.AuthedUser)
	if err != nil {
		return errors.Wrap(err, "failed to save authed user")
	}

	log.Infof("%s\n", oauthResp)

	return nil
}

func (s *Client) SendRant(sc SlashCommand, genedRant string) error {

	reply := CommandReply{
		Text:           genedRant,
		ResponseType:   inChannel,
		DeleteOriginal: true,
	}
	replyBody, err := json.Marshal(reply)
	if err != nil {
		return errors.Wrap(err, "failed to marshall reply")
	}

	req, err := http.NewRequest(http.MethodPost, sc.ResponseURL, bytes.NewReader(replyBody))
	if err != nil {
		return errors.Wrap(err, "failed to create req")
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send http req")
	}
	log.Infof("reply req: %+v", resp)

	return nil
}
