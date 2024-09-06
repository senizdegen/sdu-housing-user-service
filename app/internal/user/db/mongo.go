package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/senizdegen/sdu-housing/user-service/internal/apperror"
	"github.com/senizdegen/sdu-housing/user-service/internal/user"
	"github.com/senizdegen/sdu-housing/user-service/pkg/logging"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var _ user.Storage = &db{}

type db struct {
	collection *mongo.Collection
	logger     logging.Logger
}

func NewStorage(storage *mongo.Database, collection string, logger logging.Logger) user.Storage {
	return &db{
		collection: storage.Collection(collection),
		logger:     logger,
	}
}

func (s *db) FindOne(ctx context.Context, uuid string) (u user.User, err error) {
	objectID, err := primitive.ObjectIDFromHex(uuid)
	if err != nil {
		return u, fmt.Errorf("failed to convert hex to objectid. error: %w", err)
	}

	filter := bson.M{"_id": objectID}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result := s.collection.FindOne(ctx, filter)
	if result.Err() != nil {
		s.logger.Error(result.Err())
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return u, apperror.ErrNotFound
		}
		return u, fmt.Errorf("failed to execute query. error: %w", err)
	}
	if err = result.Decode(&u); err != nil {
		return u, fmt.Errorf("failed to decode document. error: %w", err)
	}

	return u, nil
}

func (s *db) FindByPhoneNumber(ctx context.Context, phoneNumber string) (u user.User, err error) {
	s.logger.Debug("FIND BY PHONE NUMBER")
	filter := bson.M{"phone_number": phoneNumber}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result := s.collection.FindOne(ctx, filter)
	s.logger.Trace(result)

	if err != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			s.logger.Error("error not found")
			return u, apperror.ErrNotFound
		}
		return u, fmt.Errorf("failed to execute query. error: %w", err)
	}
	if err = result.Decode(&u); err != nil {
		return u, fmt.Errorf("failed to decode document. error: %w", err)
	}

	return u, nil
}

func (s *db) Create(ctx context.Context, user user.User) (string, error) {
	nCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	result, err := s.collection.InsertOne(nCtx, user)
	if err != nil {
		return "", fmt.Errorf("failed to execute query. error: %s", err)
	}

	oid, ok := result.InsertedID.(primitive.ObjectID)
	if ok {
		return oid.Hex(), nil
	}
	return "", fmt.Errorf("failed to convert object id to hex")
}
