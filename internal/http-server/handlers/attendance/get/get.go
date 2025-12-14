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

type AttendanceGetter interface {
	GetAttendance(ctx context.Context, id string) (*api.AttendanceResponse, error)
	ListAttendance(ctx context.Context, teacherID *string, from, to *time.Time) ([]*api.AttendanceResponse, error)
}

type Response struct {
	response.Response
	Attendances []api.AttendanceResponse `json:"attendances,omitempty"`
	Attendance  *api.AttendanceResponse  `json:"attendance,omitempty"`
}

func New(log *slog.Logger, getter AttendanceGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.attendance.get.New"

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
			attendance, err := getter.GetAttendance(r.Context(), id)

			if errors.Is(err, response.ErrNotFound) {
				log.Error("resource not found")
				w.WriteHeader(http.StatusNotFound)
				render.JSON(w, r, response.Error(string(response.NOT_FOUND), "resource not found"))
				return
			}

			if err != nil {
				log.Error("Failed to get attendance", sl.Err(err))
				w.WriteHeader(http.StatusInternalServerError)
				render.JSON(w, r, response.Error(string(response.FAILED_REQUEST), "failed to get attendance"))
				return
			}

			log.Info("Attendance retrieved", slog.Any("attendance", attendance))
			responseOK(w, r, attendance)
			return
		}

		// List
		var teacherIDPtr *string
		if teacherID != "" {
			teacherIDPtr = &teacherID
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

		attendances, err := getter.ListAttendance(r.Context(), teacherIDPtr, from, to)

		if err != nil {
			log.Error("Failed to list attendance", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, response.Error(string(response.FAILED_REQUEST), "failed to list attendance"))
			return
		}

		log.Info("Attendance retrieved", slog.Int("count", len(attendances)))
		attendancesResponse := make([]api.AttendanceResponse, len(attendances))
		for i, a := range attendances {
			attendancesResponse[i] = *a
		}
		render.JSON(w, r, Response{
			Attendances: attendancesResponse,
		})
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, attendance *api.AttendanceResponse) {
	render.JSON(w, r, Response{
		Attendance: attendance,
	})
}

