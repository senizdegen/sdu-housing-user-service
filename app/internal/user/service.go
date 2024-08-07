package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cristalhq/jwt/v3"
	"github.com/google/uuid"
	"github.com/senizdegen/sdu-housing/user-service/internal/apperror"
	"github.com/senizdegen/sdu-housing/user-service/internal/config"
	"github.com/senizdegen/sdu-housing/user-service/pkg/cache"
	"github.com/senizdegen/sdu-housing/user-service/pkg/logging"
	"golang.org/x/crypto/bcrypt"
)

var _ Service = &service{}

type service struct {
	storage Storage
	logger  logging.Logger
	rtCache cache.Repository
}

func NewService(userStorage Storage, logger logging.Logger, rtCache cache.Repository) (Service, error) {
	return &service{
		storage: userStorage,
		logger:  logger,
		rtCache: rtCache,
	}, nil
}

type RT struct {
	RefreshToken string `json:"refresh_token"`
}
type UserClaims struct {
	jwt.RegisteredClaims
	UUID string `json:"uuid"`
	Role string `json:"role"`
}

type Service interface {
	GetOne(ctx context.Context, uuid string) (User, error)
	GetByPhoneNumberAndPassword(ctx context.Context, email, password string) (User, error)
	Create(ctx context.Context, dto CreateUserDTO) (string, error)
	GenerateAccessToken(u User) ([]byte, error)
	UpdateRefreshToken(rt RT) ([]byte, error)
}

func (s *service) GetOne(ctx context.Context, uuid string) (User, error) {
	u, err := s.storage.FindOne(ctx, uuid)

	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return u, err
		}
		return u, fmt.Errorf("faield to find user bu uuid. error %w", err)
	}

	return u, nil
}

func (s service) GetByPhoneNumberAndPassword(ctx context.Context, phoneNumber, password string) (u User, err error) {
	u, err = s.storage.FindByPhoneNumber(ctx, phoneNumber)

	if err != nil {

		if errors.Is(err, apperror.ErrNotFound) {

			return u, err
		}
		return u, fmt.Errorf("failed to find user by phone number. error: %w", err)
	}

	//passwords in db are hashed
	if err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		return u, apperror.ErrNotFound
	}

	s.logger.Info("Generate jwt token")
	tokenBytes, err := s.GenerateAccessToken(u)
	if err != nil {
		return u, fmt.Errorf("failed to generate token. error: %s", err)
	}

	u.JWTToken = string(tokenBytes)

	return u, nil
}

func (s *service) Create(ctx context.Context, dto CreateUserDTO) (userUUID string, err error) {
	s.logger.Debug("check password and repeated password")
	if dto.Password != dto.RepeatPassword {
		return userUUID, apperror.BadRequestError("password does not match repeated password")
	}

	user := NewUser(dto)

	s.logger.Debug("generate password hash")
	err = user.GeneratePasswordHash()
	if err != nil {
		s.logger.Errorf("failed to create user due to error: %s", err)
		return
	}

	userUUID, err = s.storage.Create(ctx, user)
	if err != nil {

		if errors.Is(err, apperror.ErrNotFound) {
			return userUUID, err
		}
		return userUUID, fmt.Errorf("failed to create user. error: %s", err)
	}
	return userUUID, nil
}

func (s *service) GenerateAccessToken(u User) ([]byte, error) {
	key := []byte(config.GetConfig().JWT.Secret)
	signer, err := jwt.NewSignerHS(jwt.HS256, key)
	if err != nil {
		return nil, err
	}
	builder := jwt.NewBuilder(signer)
	claims := UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        u.UUID,
			Audience:  []string{"users"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 60)),
		},
		UUID: u.UUID,
		Role: u.Role,
	}

	token, err := builder.Build(claims)
	if err != nil {
		return nil, err
	}

	s.logger.Info("create refresh token")
	refreshTokenUuid := uuid.New()
	userBytes, _ := json.Marshal(u)
	err = s.rtCache.Set([]byte(refreshTokenUuid.String()), userBytes, 0)
	if err != nil {
		s.logger.Error(err)
		return nil, err
	}

	jsonBytes, err := json.Marshal(map[string]string{
		"token":         token.String(),
		"refresh_token": refreshTokenUuid.String(),
	})

	if err != nil {
		return nil, err
	}

	return jsonBytes, nil
}

func (s *service) UpdateRefreshToken(rt RT) ([]byte, error) {
	defer s.rtCache.Del([]byte(rt.RefreshToken))

	userBytes, err := s.rtCache.Get([]byte(rt.RefreshToken))
	if err != nil {
		return nil, err
	}
	var u User
	err = json.Unmarshal(userBytes, &u)
	if err != nil {
		return nil, err
	}
	return s.GenerateAccessToken(u)
}
