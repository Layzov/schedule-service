package reschedule

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

type BookingRescheduler interface {
	RescheduleBooking(ctx context.Context, bookingID string, newSlotId string) (*api.BookingResponse, error)
}

type Request struct {
	api.BookingRescheduleRequest
}

type Response struct {
	response.Response
	Booking api.BookingResponse `json:"booking,omitempty"`
}

func New(log *slog.Logger, rescheduler BookingRescheduler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.bookings.reschedule.New"

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

		if req.BookingID == "" {
			log.Error("Booking_id is empty")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST), "booking_id is required"))
			return
		}

		if req.NewSlotID == "" {
			log.Error("new_slot_id is empty")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST), "new_slot_id is required"))
			return
		}

		booking, err := rescheduler.RescheduleBooking(r.Context(), req.BookingID, req.NewSlotID)

		if errors.Is(err, response.ErrNotFound) {
			log.Error("resource not found")
			w.WriteHeader(http.StatusNotFound)
			render.JSON(w, r, response.Error(string(response.NOT_FOUND), "resource not found"))
			return
		}

		if errors.Is(err, response.ErrSlotNotAvailable) {
			log.Error("slot is not available")
			w.WriteHeader(http.StatusConflict)
			render.JSON(w, r, response.Error(string(response.SLOT_NOT_AVAILABLE), "slot is not available"))
			return
		}

		if err != nil {
			log.Error("Failed to reschedule booking", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, response.Error(string(response.FAILED_REQUEST), "failed to reschedule booking"))
			return
		}

		log.Info("Booking rescheduled", slog.Any("booking", booking))
		responseOK(w, r, booking)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, booking *api.BookingResponse) {
	render.JSON(w, r, Response{
		Booking: *booking,
	})
}

