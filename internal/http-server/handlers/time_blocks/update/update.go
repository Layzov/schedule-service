package update

import (
	"rasp-service/api"
	"rasp-service/pkg/response"
	"rasp-service/pkg/sl"
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

type TimeBlockUpdater interface {
	UpdateTimeBlock(ctx context.Context, id string, req *api.TimeBlockRequest) (*api.TimeBlockResponse, error)
}

type Request struct {
	api.TimeBlockRequest
}

type Response struct {
	response.Response
	TimeBlock api.TimeBlockResponse `json:"time_block,omitempty"`
}

func New(log *slog.Logger, updater TimeBlockUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.time_blocks.update.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		id := chi.URLParam(r, "id")
		if id == "" {
			log.Error("id is empty")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST), "id is required"))
			return
		}

		var req Request

		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("Failed to decode request body", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST), "failed to decode request"))
			return
		}

		log.Info("Request body decoded", slog.Any("request", req))

		timeBlock, err := updater.UpdateTimeBlock(r.Context(), id, &req.TimeBlockRequest)

		if errors.Is(err, response.ErrNotFound) {
			log.Error("resource not found")
			w.WriteHeader(http.StatusNotFound)
			render.JSON(w, r, response.Error(string(response.NOT_FOUND), "resource not found"))
			return
		}

		if err != nil {
			log.Error("Failed to update time block", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, response.Error(string(response.FAILED_REQUEST), "failed to update time block"))
			return
		}

		log.Info("Time block updated", slog.Any("time_block", timeBlock))
		responseOK(w, r, timeBlock)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, timeBlock *api.TimeBlockResponse) {
	render.JSON(w, r, Response{
		TimeBlock: *timeBlock,
	})
}

