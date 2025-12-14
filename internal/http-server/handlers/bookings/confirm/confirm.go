package confirm

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

type BookingConfirmer interface {
	ConfirmBooking(ctx context.Context, bookingID string) (*api.BookingResponse, error)
}

type Response struct {
	response.Response
	Booking api.BookingResponse `json:"booking,omitempty"`
}

func New(log *slog.Logger, confirmer BookingConfirmer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.bookings.confirm.New"

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

		booking, err := confirmer.ConfirmBooking(r.Context(), id)

		if errors.Is(err, response.ErrNotFound) {
			log.Error("resource not found")
			w.WriteHeader(http.StatusNotFound)
			render.JSON(w, r, response.Error(string(response.NOT_FOUND), "resource not found"))
			return
		}

		if err != nil {
			log.Error("Failed to confirm booking", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, response.Error(string(response.FAILED_REQUEST), "failed to confirm booking"))
			return
		}

		log.Info("Booking confirmed", slog.Any("booking", booking))
		responseOK(w, r, booking)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, booking *api.BookingResponse) {
	render.JSON(w, r, Response{
		Booking: *booking,
	})
}

