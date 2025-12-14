package get

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"rasp-service/api"
	"rasp-service/internal/service"
	"rasp-service/pkg/response"
	"rasp-service/pkg/sl"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type SlotGetter interface {
	GetSlot(ctx context.Context, id string) (*api.SlotResponse, error)
	ListSlots(ctx context.Context, filters *service.SlotFilters) ([]*api.SlotResponse, error)
	GetSlotsByIDs(ctx context.Context, ids []string) ([]*api.SlotResponse, error)
}

type Response struct {
	response.Response
	Slots []api.SlotResponse `json:"slots,omitempty"`
	Slot  *api.SlotResponse  `json:"slot,omitempty"`
}

func New(log *slog.Logger, getter SlotGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.slots.get.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		if r.URL.Path == "/slots/batch" {
			idsStr := r.URL.Query().Get("ids")
			if idsStr != "" {
				ids := strings.Split(idsStr, ",")
				slots, err := getter.GetSlotsByIDs(r.Context(), ids)
				if err != nil {
					log.Error("Failed to get slots by IDs", sl.Err(err))
					w.WriteHeader(http.StatusInternalServerError)
					render.JSON(w, r, response.Error(string(response.FAILED_REQUEST), "failed to get slots"))
					return
				}
				slotsResponse := make([]api.SlotResponse, len(slots))
				for i, s := range slots {
					slotsResponse[i] = *s
				}
				render.JSON(w, r, Response{Slots: slotsResponse})
				return
			}
		}

		id := chi.URLParam(r, "id")

		if id != "" {
			// Get by ID
			slot, err := getter.GetSlot(r.Context(), id)

			if errors.Is(err, response.ErrNotFound) {
				log.Error("resource not found")
				w.WriteHeader(http.StatusNotFound)
				render.JSON(w, r, response.Error(string(response.NOT_FOUND), "resource not found"))
				return
			}

			if err != nil {
				log.Error("Failed to get slot", sl.Err(err))
				w.WriteHeader(http.StatusInternalServerError)
				render.JSON(w, r, response.Error(string(response.FAILED_REQUEST), "failed to get slot"))
				return
			}

			log.Info("Slot retrieved", slog.Any("slot", slot))
			responseOK(w, r, slot)
			return
		}

		// List with filters
		filters := &service.SlotFilters{}

		if teacherID := r.URL.Query().Get("teacher_id"); teacherID != "" {
			filters.TeacherID = &teacherID
		}

		if fromStr := r.URL.Query().Get("from"); fromStr != "" {
			if t, err := time.Parse("2006-01-02", fromStr); err == nil {
				filters.From = &t
			} else if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
				filters.From = &t
			}
		}

		if toStr := r.URL.Query().Get("to"); toStr != "" {
			if t, err := time.Parse("2006-01-02", toStr); err == nil {
				filters.To = &t
			} else if t, err := time.Parse(time.RFC3339, toStr); err == nil {
				filters.To = &t
			}
		}

		if durationStr := r.URL.Query().Get("duration"); durationStr != "" {
			if duration, err := strconv.Atoi(durationStr); err == nil {
				filters.Duration = &duration
			}
		}

		if status := r.URL.Query().Get("status"); status != "" {
			filters.Status = &status
		}

		if q := r.URL.Query().Get("q"); q != "" {
			filters.Q = &q
		}

		if pageStr := r.URL.Query().Get("page"); pageStr != "" {
			if page, err := strconv.Atoi(pageStr); err == nil {
				filters.Page = &page
			}
		}

		if perPageStr := r.URL.Query().Get("per_page"); perPageStr != "" {
			if perPage, err := strconv.Atoi(perPageStr); err == nil {
				filters.PerPage = &perPage
			}
		}

		if sort := r.URL.Query().Get("sort"); sort != "" {
			filters.Sort = &sort
		}

		slots, err := getter.ListSlots(r.Context(), filters)

		if err != nil {
			log.Error("Failed to list slots", sl.Err(err))
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, response.Error(string(response.FAILED_REQUEST), "failed to list slots"))
			return
		}

		log.Info("Slots retrieved", slog.Int("count", len(slots)))
		slotsResponse := make([]api.SlotResponse, len(slots))
		for i, s := range slots {
			slotsResponse[i] = *s
		}
		render.JSON(w, r, Response{
			Slots: slotsResponse,
		})
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, slot *api.SlotResponse) {
	render.JSON(w, r, Response{
		Slot: slot,
	})
}
