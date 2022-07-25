package rant_store

import (
	"context"

	"github.com/BadgerBadgerBadgerBadger/rant/internal/rant/slack"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const collectionAuthedUser = "authedUser"

type store struct {
	config      Config
	mongoClient *mongo.Client
}

type Config struct {
	Database string `json:"database" envconfig:"DATABASE_DATABASE"`
	ConnUri  string `json:"conn_uri" envconfig:"DATABASE_CONN_URI"`
}

func NewRantStore(c Config) (slack.Store, error) {

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(c.ConnUri))
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to mongodb")
	}

	return &store{
		config:      c,
		mongoClient: client,
	}, nil
}

func (s *store) GetAuthedUser(userID string) (slack.AuthedUser, bool, error) {

	authedUserCollection := s.mongoClient.Database(s.config.Database).Collection(collectionAuthedUser)

	filter := bson.D{
		{Key: "ID", Value: userID},
	}

	result := authedUserCollection.FindOne(context.Background(), filter)
	if result.Err() != nil {

		if result.Err() == mongo.ErrNoDocuments {
			return slack.AuthedUser{}, false, nil
		}

		return slack.AuthedUser{}, false, errors.Wrap(result.Err(), "failed to fetch authed user")
	}

	authedUser := slack.AuthedUser{}
	err := result.Decode(&authedUser)
	if err != nil {
		return slack.AuthedUser{}, false, errors.Wrap(err, "failed to decode authed user")
	}

	return authedUser, true, nil
}

func (s *store) StoreAuthedUser(userID string, authedUser slack.AuthedUser) error {

	authedUserCollection := s.mongoClient.Database(s.config.Database).Collection(collectionAuthedUser)

	_, err := authedUserCollection.InsertOne(context.Background(), authedUser)
	if err != nil {
		return errors.Wrap(err, "failed to write")
	}

	return nil
}
