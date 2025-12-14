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

type AttendanceCreator interface {
	CreateAttendance(ctx context.Context, req *api.AttendanceRequest) (*api.AttendanceResponse, error)
}

type Request struct {
	api.AttendanceRequest
}

type Response struct {
	response.Response
	Attendance api.AttendanceResponse `json:"attendance,omitempty"`
}

func New(log *slog.Logger, creator AttendanceCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.attendance.create.New"

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

		if req.BookingID == "" {
			log.Error("booking_id is empty")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST), "booking_id is required"))
			return
		}

		if req.Status == "" {
			log.Error("status is empty")
			w.WriteHeader(http.StatusBadRequest)
			render.JSON(w, r, response.Error(string(response.BAD_REQUEST), "status is required"))
			return
		}

		attendance, err := creator.CreateAttendance(r.Context(), &req.AttendanceRequest)

		if errors.Is(err, response.ErrNotFound) {
			log.Error("resource not found")
			w.WriteHeader(http.StatusNotFound)
			render.JSON(w, r, response.Error(string(response.NOT_FOUND), "resource not found"))
			return
		}

		if err != nil {
			log.Error("Failed to create attendance", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, response.Error(string(response.FAILED_REQUEST), "failed to create attendance"))
			return
		}

		log.Info("Attendance created", slog.Any("attendance", attendance))

		w.WriteHeader(http.StatusCreated)
		responseOK(w, r, attendance)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, attendance *api.AttendanceResponse) {
	render.JSON(w, r, Response{
		Attendance: *attendance,
	})
}

