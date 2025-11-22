package set

import (
	"avito-test-assignment-backend/internal/models"
	"avito-test-assignment-backend/pkg/response"
	"avito-test-assignment-backend/pkg/sl"
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

type UserSetter interface {
	SetIsActiveService(ctx context.Context, userID string, isActive bool) (*models.User, error)
}

type Request struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

type Response struct {
	response.Response
	models.User `json:"user,omitempty"`
}

func New(log *slog.Logger, userSetter UserSetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.teams.set.set.New"
		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("Failed to decode request body", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST),"failed to decode request"))

			return
		}

		log.Info("Request body decoded", slog.Any("request", req))

		if req.UserID == ""{
			log.Error("user_id is empty")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST),"user_id is empty"))

			return
		}

		user, err := userSetter.SetIsActiveService(r.Context(), req.UserID, req.IsActive)

		if errors.Is(err, response.ErrNotFound) {
			log.Error("team_name not found")
			w.WriteHeader(http.StatusNotFound)
			render.JSON(w, r, response.Error(string(response.NOT_FOUND),"resource not found"))

			return
		}

		if err != nil {
			log.Error("Failed to set user status", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, response.Error(string(response.FAILED_REQUEST),"failed to set user status"))

			return
		}

		log.Info("User status was set ", slog.Any("user", user))

		responseOK(w, r, user)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, user *models.User) {
	render.JSON(w, r, Response{
		User: *user,
	})
}