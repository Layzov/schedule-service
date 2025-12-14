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

type BookingCreator interface {
	CreateBooking(ctx context.Context, req *api.BookingRequest, idempotencyKey *string) (*api.BookingResponse, error)
}

type Request struct {
	api.BookingRequest
}

type Response struct {
	response.Response
	Booking api.BookingResponse `json:"booking,omitempty"`
}

func New(log *slog.Logger, creator BookingCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.bookings.create.New"

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

		if req.SlotID == "" {
			log.Error("slot_id is empty")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST), "slot_id is required"))
			return
		}

		if req.StudentID == "" {
			log.Error("student_id is empty")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST), "student_id is required"))
			return
		}

		idempotencyKey := r.Header.Get("Idempotency-Key")
		var idempotencyKeyPtr *string
		if idempotencyKey != "" {
			idempotencyKeyPtr = &idempotencyKey
		}

		booking, err := creator.CreateBooking(r.Context(), &req.BookingRequest, idempotencyKeyPtr)

		if errors.Is(err, response.ErrLocked) {
			log.Error("resource is locked")
			w.WriteHeader(http.StatusLocked)
			render.JSON(w, r, response.Error(string(response.LOCKED), "resource is locked"))
			return
		}

		if errors.Is(err, response.ErrSlotNotAvailable) {
			log.Error("slot is not available")
			w.WriteHeader(http.StatusConflict)
			render.JSON(w, r, response.Error(string(response.SLOT_NOT_AVAILABLE), "slot is not available"))
			return
		}

		if errors.Is(err, response.ErrNotFound) {
			log.Error("resource not found")
			w.WriteHeader(http.StatusNotFound)
			render.JSON(w, r, response.Error(string(response.NOT_FOUND), "resource not found"))
			return
		}

		if err != nil {
			log.Error("Failed to create booking", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, response.Error(string(response.FAILED_REQUEST), "failed to create booking"))
			return
		}

		log.Info("Booking created", slog.Any("booking", booking))

		w.WriteHeader(http.StatusCreated)
		responseOK(w, r, booking)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, booking *api.BookingResponse) {
	render.JSON(w, r, Response{
		Booking: *booking,
	})
}

