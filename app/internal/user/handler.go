package user

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/senizdegen/sdu-housing/user-service/internal/apperror"
	"github.com/senizdegen/sdu-housing/user-service/pkg/logging"
)

const (
	usersURL = "/api/users"
	userURL  = "/api/users/:uuid"
)

type Handler struct {
	Logger      logging.Logger
	UserService Service
}

func (h *Handler) Register(router *httprouter.Router) {
	router.HandlerFunc(http.MethodGet, userURL, apperror.Middleware(h.GetUser))
	router.HandlerFunc(http.MethodGet, usersURL, apperror.Middleware(h.GetUserByPhoneNumberAndPassword))
	router.HandlerFunc(http.MethodPost, usersURL, apperror.Middleware(h.CreateUser))
}

/*
В MongoDB ObjectID представляет собой 12-байтовый идентификатор,
который обычно представлен в виде 24-символьной шестнадцатеричной строки.
ПРИМЕР: "507f1f77bcf86cd799439011".
*/
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) error {
	h.Logger.Info("GET USER")
	w.Header().Set("Content-Type", "application/json")

	h.Logger.Debug("get uuid from context")
	params := r.Context().Value(httprouter.ParamsKey).(httprouter.Params)
	userUUID := params.ByName("uuid")

	user, err := h.UserService.GetOne(r.Context(), userUUID)
	if err != nil {
		return err
	}
	h.Logger.Debug("marshal user")
	userBytes, err := json.Marshal(user)

	if err != nil {
		return fmt.Errorf("failed to marshall user. error: %w", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(userBytes)

	return nil
}

func (h *Handler) GetUserByPhoneNumberAndPassword(w http.ResponseWriter, r *http.Request) error {
	h.Logger.Info("GET USER BY PHONE NUMBER AND PASSWORD")
	w.Header().Set("Content-Type", "application/json")

	h.Logger.Debug("get phone number and password from URL")
	phoneNumber := r.URL.Query().Get("phone_number")
	password := r.URL.Query().Get("password")
	if phoneNumber == "" || password == "" {
		return apperror.BadRequestError("invalid query parameters email or password")
	}

	h.Logger.Debugf("phone number: %s password: %s", phoneNumber, password)

	user, err := h.UserService.GetByPhoneNumberAndPassword(r.Context(), phoneNumber, password)
	if err != nil {
		return err
	}

	h.Logger.Debug("marshal user")

	userBytes, err := json.Marshal(user)
	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusOK)
	w.Write(userBytes)

	return nil
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) error {
	h.Logger.Info("CREATE USER")
	w.Header().Set("Content-Type", "application/json")

	h.Logger.Debug("decode create user dto")
	var crUser CreateUserDTO
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&crUser); err != nil {
		return apperror.BadRequestError("invalid JSON scheme. check swagger API")
	}
	userUUID, err := h.UserService.Create(r.Context(), crUser)
	if err != nil {
		return err
	}

	w.Header().Set("Location", fmt.Sprintf("%s/%s", usersURL, userUUID))
	w.WriteHeader(http.StatusCreated)

	return nil
}
