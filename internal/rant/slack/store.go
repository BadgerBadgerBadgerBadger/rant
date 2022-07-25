package slack

type AuthedUser struct {
	ID          string `json:"id"`
	Scope       string `json:"scope"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type Store interface {
	GetAuthedUser(userID string) (AuthedUser, bool, error)
	StoreAuthedUser(userID string, authedUser AuthedUser) error
}
