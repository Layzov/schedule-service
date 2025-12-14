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

type AvailabilityTemplateCreator interface {
	CreateAvailabilityTemplate(ctx context.Context, req *api.AvailabilityTemplateRequest) (*api.AvailabilityTemplateResponse, error)
}

type Request struct {
	api.AvailabilityTemplateRequest
}

type Response struct {
	response.Response
	Template api.AvailabilityTemplateResponse `json:"template,omitempty"`
}

func New(log *slog.Logger, creator AvailabilityTemplateCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.availability_templates.create.New"

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

		template, err := creator.CreateAvailabilityTemplate(r.Context(), &req.AvailabilityTemplateRequest)

		if errors.Is(err, response.ErrNotFound) {
			log.Error("resource not found")
			w.WriteHeader(http.StatusNotFound)
			render.JSON(w, r, response.Error(string(response.NOT_FOUND), "resource not found"))
			return
		}

		if err != nil {
			log.Error("Failed to create availability template", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, response.Error(string(response.FAILED_REQUEST), "failed to create availability template"))
			return
		}

		log.Info("Availability template created", slog.Any("template", template))

		w.WriteHeader(http.StatusCreated)
		responseOK(w, r, template)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, template *api.AvailabilityTemplateResponse) {
	render.JSON(w, r, Response{
		Template: *template,
	})
}

