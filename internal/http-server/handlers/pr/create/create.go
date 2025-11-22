package create

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

type PRCreator interface {
	CreatePullRequestService(ctx context.Context, pr *api.PRCreateRequest) (*api.PullRequest, error)
}

type Request struct {
	api.PRCreateRequest
}

type Response struct {
	response.Response
	PR api.PullRequest `json:"pr,omitempty"`
}

func New(log *slog.Logger, prCreator PRCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.pr.create.create.New"

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

		if req.PullRequestID == ""{
			log.Error("PR id is empty")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST),"PR id is empty"))

			return
		}
		if req.AuthorID == ""{
			log.Error("author_id is empty")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST),"author_id is empty"))

			return
		}

		log.Info("PR_REQ ", slog.Any("pr", req.PRCreateRequest))

		pr, err := prCreator.CreatePullRequestService(r.Context(), &req.PRCreateRequest)

		log.Info("PR_final", slog.Any("pr", pr))

		if errors.Is(err, response.ErrPRExists) {
			log.Error("PR id already exists", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.PR_EXISTS),"PR id already exists"))

			return
		}
		if errors.Is(err, response.ErrNotFound) {
			log.Error("resource not found")
			w.WriteHeader(http.StatusNotFound)
			render.JSON(w, r, response.Error(string(response.NOT_FOUND),"resource not found"))

			return
		}

		if err != nil {
			log.Error("Failed to create PR", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, response.Error(string(response.FAILED_REQUEST),"failed to create PR"))

			return
		}

		log.Info("PR was created ", slog.Any("pr", pr))

		responseOK(w, r, pr)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, pr *api.PullRequest) {
	render.JSON(w, r, Response{
		PR: *pr,
	})
}