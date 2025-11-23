package merge

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

type PRMerger interface {
	MergePRService(ctx context.Context, prID string) (*api.PullRequest, error)
}

type Request struct {
	PrID string `json:"pull_request_id"`
}

type Response struct {
	response.Response
	PR api.PullRequest `json:"pr,omitempty"`
}

func New(log *slog.Logger, prMerger PRMerger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.pr.merge.merge.New"

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
		
		if req.PrID == ""{
			log.Error("PR id is empty")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST),"PR id is empty"))

			return
		}

		pr, err := prMerger.MergePRService(r.Context(), req.PrID)
		if errors.Is(err, response.ErrNotFound) {
			log.Error("resource not found")
			w.WriteHeader(http.StatusNotFound)
			render.JSON(w, r, response.Error(string(response.NOT_FOUND),"resource not found"))

			return
		}

		if err != nil {
			log.Error("Failed to merge PR", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, response.Error(string(response.FAILED_REQUEST),"failed to merge PR"))

			return
		}

		log.Info("PR was merged ", slog.Any("pr", pr))

		responseOK(w, r, pr)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, pr *api.PullRequest) {
	render.JSON(w, r, Response{
		PR: *pr,
	})
}