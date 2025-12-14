package generate

import (
	"rasp-service/api"
	"rasp-service/pkg/response"
	"rasp-service/pkg/sl"
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

type SlotGenerator interface {
	GenerateSlots(ctx context.Context, req *api.SlotGenerateRequest) (string, error)
}

type Request struct {
	api.SlotGenerateRequest
}

type Response struct {
	response.Response
	JobID string `json:"job_id"`
}

func New(log *slog.Logger, generator SlotGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.slots.generate.New"

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

		if req.TemplateID == nil && req.TeacherID == nil {
			log.Error("template_id or teacher_id is required")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST), "template_id or teacher_id is required"))
			return
		}

		jobID, err := generator.GenerateSlots(r.Context(), &req.SlotGenerateRequest)

		if err != nil {
			log.Error("Failed to generate slots", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, response.Error(string(response.FAILED_REQUEST), "failed to generate slots"))
			return
		}

		log.Info("Slots generation started", slog.String("job_id", jobID))

		w.WriteHeader(http.StatusAccepted)
		render.JSON(w, r, Response{
			JobID: jobID,
		})
	}
}

