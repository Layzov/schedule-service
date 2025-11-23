package reassign

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

type PRReassigner interface {
	ReassignPRReviewersService(ctx context.Context, prID string, oldReviewerID string) (*api.PullRequest, error)
}

type Request struct {
	PrID string `json:"pull_request_id"`
	OldReviewerID string `json:"old_reviewer_id"`
}

type Response struct {
	response.Response
	PR api.PullRequest
	ReplacedBY string `json:"replaced_by"`
}

func New(log *slog.Logger, reasigner PRReassigner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request){
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
		if req.OldReviewerID == ""{
			log.Error("Old reviewer id is empty")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST),"Old reviewer id is empty"))

			return
		}

		pr, err := reasigner.ReassignPRReviewersService(r.Context(), req.PrID, req.OldReviewerID)
		if errors.Is(err, response.ErrNotFound) {
			log.Error("resource not found")
			w.WriteHeader(http.StatusNotFound)
			render.JSON(w, r, response.Error(string(response.NOT_FOUND),"resource not found"))

			return
		}
		if errors.Is(err, response.ErrPRMerged) {
			log.Error("cannot reassign on merged PR")
			w.WriteHeader(http.StatusConflict)
			render.JSON(w, r, response.Error(string(response.PR_MERGED),"cannot reassign on merged PR"))

			return
		}
		if errors.Is(err, response.ErrNoCandidate) {
			log.Error("no active replacement candidate in team")
			w.WriteHeader(http.StatusConflict)
			render.JSON(w, r, response.Error(string(response.NO_CANDIDATE),"no active replacement candidate in team"))

			return
		}
		if errors.Is(err, response.ErrNotAssigned) {
			log.Error("reviewer is not assigned to this PR")
			w.WriteHeader(http.StatusConflict)
			render.JSON(w, r, response.Error(string(response.NOT_ASSIGNED),"reviewer is not assigned to this PR"))

			return
		}
		if err != nil {
			log.Error("Failed to reassign", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, response.Error(string(response.FAILED_REQUEST),"Failed to reassign"))

			return
		}

		log.Info("Reviewer was reassigned ", slog.Any("pr", pr))


		responseOK(w, r, pr, pr.Reviewers[0])
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, pr *api.PullRequest, newReviewer string) {
	render.JSON(w, r, Response{
		PR: *pr,
		ReplacedBY: newReviewer,
	})
}