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

type BookingGetter interface {
	GetBooking(ctx context.Context, id string) (*api.BookingResponse, error)
	ListBookings(ctx context.Context, studentID, teacherID *string, from, to *time.Time, status *string) ([]*api.BookingResponse, error)
}

type Response struct {
	response.Response
	Bookings []api.BookingResponse `json:"bookings,omitempty"`
	Booking  *api.BookingResponse  `json:"booking,omitempty"`
}

func New(log *slog.Logger, getter BookingGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.bookings.get.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		id := chi.URLParam(r, "id")

		if id != "" {
			// Get by ID
			booking, err := getter.GetBooking(r.Context(), id)

			if errors.Is(err, response.ErrNotFound) {
				log.Error("resource not found")
				w.WriteHeader(http.StatusNotFound)
				render.JSON(w, r, response.Error(string(response.NOT_FOUND), "resource not found"))
				return
			}

			if err != nil {
				log.Error("Failed to get booking", sl.Err(err))
				w.WriteHeader(http.StatusInternalServerError)
				render.JSON(w, r, response.Error(string(response.FAILED_REQUEST), "failed to get booking"))
				return
			}

			log.Info("Booking retrieved", slog.Any("booking", booking))
			responseOK(w, r, booking)
			return
		}

		// List
		studentID := r.URL.Query().Get("student_id")
		teacherID := r.URL.Query().Get("teacher_id")
		fromStr := r.URL.Query().Get("from")
		toStr := r.URL.Query().Get("to")
		status := r.URL.Query().Get("status")

		var studentIDPtr, teacherIDPtr, statusPtr *string
		if studentID != "" {
			studentIDPtr = &studentID
		}
		if teacherID != "" {
			teacherIDPtr = &teacherID
		}
		if status != "" {
			statusPtr = &status
		}

		var from, to *time.Time
		if fromStr != "" {
			if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
				from = &t
			} else if t, err := time.Parse("2006-01-02", fromStr); err == nil {
				from = &t
			}
		}
		if toStr != "" {
			if t, err := time.Parse(time.RFC3339, toStr); err == nil {
				to = &t
			} else if t, err := time.Parse("2006-01-02", toStr); err == nil {
				to = &t
			}
		}

		bookings, err := getter.ListBookings(r.Context(), studentIDPtr, teacherIDPtr, from, to, statusPtr)

		if err != nil {
			log.Error("Failed to list bookings", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, response.Error(string(response.FAILED_REQUEST), "failed to list bookings"))
			return
		}

		log.Info("Bookings retrieved", slog.Int("count", len(bookings)))
		bookingsResponse := make([]api.BookingResponse, len(bookings))
		for i, b := range bookings {
			bookingsResponse[i] = *b
		}
		render.JSON(w, r, Response{
			Bookings: bookingsResponse,
		})
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, booking *api.BookingResponse) {
	render.JSON(w, r, Response{
		Booking: booking,
	})
}

