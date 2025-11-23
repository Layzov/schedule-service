package reviews

import (
	"avito-test-assignment-backend/api"
	"avito-test-assignment-backend/pkg/response"
	"avito-test-assignment-backend/pkg/sl"
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

type ReviewGetter interface {
	GetReviewService(ctx context.Context, userID string) (*[]api.PullRequestShortDto, error)
}

type Response struct {
	response.Response
	UserID string `json:"user_id,omitempty"`
	UserReviews []api.PullRequestShortDto `json:"pull_requests"`
}

func New(log *slog.Logger,reviewGetter ReviewGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.teams.get.get.New"
		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		userID := r.URL.Query().Get("user_id")
		if userID == ""{
			log.Error("Request body is empty")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST),"request body is empty"))
		}

		reviews, err :=  reviewGetter.GetReviewService(r.Context(), userID)

		if errors.Is(err, response.ErrNotFound) {
			log.Error("team_name not found")
			w.WriteHeader(http.StatusNotFound)
			render.JSON(w, r, response.Error(string(response.NOT_FOUND),"resource not found"))

			return
		}
		if err != nil {
			log.Error("Failed to get reviews", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, response.Error(string(response.FAILED_REQUEST),"failed to get reviews"))

			return
		}

		log.Info("Reviews were received ", slog.Any("reviews", reviews))

		responseOK(w, r, userID, reviews)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, userID string, reviews *[]api.PullRequestShortDto) {
	render.JSON(w, r, Response{
		UserID: userID,
		UserReviews: *reviews,
	})
}