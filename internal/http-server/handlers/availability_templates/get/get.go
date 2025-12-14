package get

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

type AvailabilityTemplateGetter interface {
	GetAvailabilityTemplate(ctx context.Context, id string) (*api.AvailabilityTemplateResponse, error)
}

type Response struct {
	response.Response
	Template api.AvailabilityTemplateResponse `json:"template,omitempty"`
}

func New(log *slog.Logger, getter AvailabilityTemplateGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.availability_templates.get.New"

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

		template, err := getter.GetAvailabilityTemplate(r.Context(), id)

		if errors.Is(err, response.ErrNotFound) {
			log.Error("resource not found")
			w.WriteHeader(http.StatusNotFound)
			render.JSON(w, r, response.Error(string(response.NOT_FOUND), "resource not found"))
			return
		}

		if err != nil {
			log.Error("Failed to get availability template", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, response.Error(string(response.FAILED_REQUEST), "failed to get availability template"))
			return
		}

		log.Info("Availability template retrieved", slog.Any("template", template))
		responseOK(w, r, template)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, template *api.AvailabilityTemplateResponse) {
	render.JSON(w, r, Response{
		Template: *template,
	})
}

