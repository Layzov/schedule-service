package get

import (
	"rasp-service/api"
	"rasp-service/pkg/response"
	"rasp-service/pkg/sl"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

type TimeBlockGetter interface {
	GetTimeBlock(ctx context.Context, id string) (*api.TimeBlockResponse, error)
	ListTimeBlocks(ctx context.Context, teacherID *string, from, to *time.Time) ([]*api.TimeBlockResponse, error)
}

type Response struct {
	response.Response
	TimeBlocks []api.TimeBlockResponse `json:"time_blocks,omitempty"`
	TimeBlock  *api.TimeBlockResponse  `json:"time_block,omitempty"`
}

func New(log *slog.Logger, getter TimeBlockGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.time_blocks.get.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		id := chi.URLParam(r, "id")
		teacherID := r.URL.Query().Get("teacher_id")
		fromStr := r.URL.Query().Get("from")
		toStr := r.URL.Query().Get("to")

		if id != "" {
			// Get by ID
			timeBlock, err := getter.GetTimeBlock(r.Context(), id)

			if errors.Is(err, response.ErrNotFound) {
				log.Error("resource not found")
				w.WriteHeader(http.StatusNotFound)
				render.JSON(w, r, response.Error(string(response.NOT_FOUND), "resource not found"))
				return
			}

			if err != nil {
				log.Error("Failed to get time block", sl.Err(err))
				w.WriteHeader(http.StatusInternalServerError)
				render.JSON(w, r, response.Error(string(response.FAILED_REQUEST), "failed to get time block"))
				return
			}

			log.Info("Time block retrieved", slog.Any("time_block", timeBlock))
			responseOK(w, r, timeBlock)
			return
		}

		// List
		var teacherIDPtr *string
		if teacherID != "" {
			teacherIDPtr = &teacherID
		}

		var from, to *time.Time
		if fromStr != "" {
			t, err := time.Parse(time.RFC3339, fromStr)
			if err == nil {
				from = &t
			}
		}
		if toStr != "" {
			t, err := time.Parse(time.RFC3339, toStr)
			if err == nil {
				to = &t
			}
		}

		timeBlocks, err := getter.ListTimeBlocks(r.Context(), teacherIDPtr, from, to)

		if err != nil {
			log.Error("Failed to list time blocks", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, response.Error(string(response.FAILED_REQUEST), "failed to list time blocks"))
			return
		}

		log.Info("Time blocks retrieved", slog.Int("count", len(timeBlocks)))
		timeBlocksResponse := make([]api.TimeBlockResponse, len(timeBlocks))
		for i, tb := range timeBlocks {
			timeBlocksResponse[i] = *tb
		}
		render.JSON(w, r, Response{
			TimeBlocks: timeBlocksResponse,
		})
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, timeBlock *api.TimeBlockResponse) {
	render.JSON(w, r, Response{
		TimeBlock: timeBlock,
	})
}

