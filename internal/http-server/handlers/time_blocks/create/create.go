package create

import (
	"rasp-service/api"
	"rasp-service/pkg/response"
	"rasp-service/pkg/sl"
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

type TimeBlockCreator interface {
	CreateTimeBlock(ctx context.Context, req *api.TimeBlockRequest) (*api.TimeBlockResponse, error)
}

type Request struct {
	api.TimeBlockRequest
}

type Response struct {
	response.Response
	TimeBlock api.TimeBlockResponse `json:"time_block,omitempty"`
}

func New(log *slog.Logger, creator TimeBlockCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.time_blocks.create.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("Failed to decode request body", sl.Err(err))
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST), "failed to decode request"))
			return
		}

		log.Info("Request body decoded", slog.Any("request", req))

		if req.TeacherID == "" {
			log.Error("teacher_id is empty")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST), "teacher_id is required"))
			return
		}

		timeBlock, err := creator.CreateTimeBlock(r.Context(), &req.TimeBlockRequest)

		if errors.Is(err, response.ErrNotFound) {
			log.Error("resource not found")
			w.WriteHeader(http.StatusNotFound)
			render.JSON(w, r, response.Error(string(response.NOT_FOUND), "resource not found"))
			return
		}

		if err != nil {
			log.Error("Failed to create time block", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, response.Error(string(response.FAILED_REQUEST), "failed to create time block"))
			return
		}

		log.Info("Time block created", slog.Any("time_block", timeBlock))

		w.WriteHeader(http.StatusCreated)
		responseOK(w, r, timeBlock)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, timeBlock *api.TimeBlockResponse) {
	render.JSON(w, r, Response{
		TimeBlock: *timeBlock,
	})
}

