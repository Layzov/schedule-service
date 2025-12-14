package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"rasp-service/api"
	"rasp-service/internal/lock"
	"rasp-service/internal/models"
	"rasp-service/pkg/response"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	store Store
	locker lock.Locker
}

func NewService(store Store, locker lock.Locker) *Service {
	return &Service{store: store, locker: locker}
}

type Store interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)

	// Availability Templates
	CreateAvailabilityTemplate(ctx context.Context, template *models.AvailabilityTemplate) (string, error)
	GetAvailabilityTemplate(ctx context.Context, id string) (*models.AvailabilityTplSlot, error)
	UpdateAvailabilityTemplate(ctx context.Context, template *models.AvailabilityTplSlot) error
	DeleteAvailabilityTemplate(ctx context.Context, id string) error

	// Time Blocks
	CreateTimeBlock(ctx context.Context, block *models.TimeBlock) (string, error)
	GetTimeBlock(ctx context.Context, id string) (*models.TimeBlock, error)
	ListTimeBlocks(ctx context.Context, teacherID *string, from, to *time.Time) ([]*models.TimeBlock, error)
	UpdateTimeBlock(ctx context.Context, block *models.TimeBlock) error
	DeleteTimeBlock(ctx context.Context, id string) error

	// Slots
	GetSlot(ctx context.Context, id string) (*models.Slot, error)
	GetSlotsByIDs(ctx context.Context, ids []string) ([]*models.Slot, error)
	ListSlots(ctx context.Context, filters interface{}) ([]*models.Slot, error)
	CreateSlot(ctx context.Context, tx *sql.Tx, slot *models.Slot) (string, error)
	UpdateSlotStatus(ctx context.Context, slotID string, status models.SlotStatus, bookingID *string) error
	GetSlotForBooking(ctx context.Context, slotID string) (*models.Slot, error)

	// Bookings
	CreateBooking(ctx context.Context, tx *sql.Tx, booking *models.Booking) (string, error)
	GetBooking(ctx context.Context, id string) (*models.Booking, error)
	ListBookings(ctx context.Context, studentID, teacherID *string, from, to *time.Time, status *string) ([]*models.Booking, error)
	UpdateBookingStatus(ctx context.Context, bookingID string, status models.BookingStatus) error
	RescheduleBooking(ctx context.Context, tx *sql.Tx, bookingID, newSlotID string) error
	DeleteBooking(ctx context.Context, bookingID string) error

	// Attendance
	CreateAttendance(ctx context.Context, attendance *models.Attendance) (string, error)
	GetAttendance(ctx context.Context, id string) (*models.Attendance, error)
	ListAttendance(ctx context.Context, teacherID *string, from, to *time.Time) ([]*models.Attendance, error)
}

type SlotFilters struct {
	TeacherID *string
	From      *time.Time
	To        *time.Time
	Duration  *int
	Status    *string
	Q         *string
	Page      *int
	PerPage   *int
	Sort      *string
}

// Availability Templates

func (s *Service) CreateAvailabilityTemplate(ctx context.Context, req *api.AvailabilityTemplateRequest) (*api.AvailabilityTemplateResponse, error) {
	const op = "service.CreateAvailabilityTemplate"

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid start_date: %w", op, err)
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid end_date: %w", op, err)
	}

	startTime, err := time.Parse("15:04", req.Recurrence.StartTime)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid start_time: %w", op, err)
	}

	endTime, err := time.Parse("15:04", req.Recurrence.EndTime)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid end_time: %w", op, err)
	}

	template := &models.AvailabilityTemplate{
		TeacherID:           req.TeacherID,
		RecurrenceDays:      req.Recurrence.Days,
		RecurrenceStartTime: startTime.Format("15:04:05"),
		RecurrenceEndTime:   endTime.Format("15:04:05"),
		SlotDurationMinutes: req.SlotDurationMinutes,
		StartDate:           startDate,
		EndDate:             endDate,
		Enabled:             req.Enabled,
	}

	id, err := s.store.CreateAvailabilityTemplate(ctx, template)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return s.GetAvailabilityTemplate(ctx, id)
}

func (s *Service) GetAvailabilityTemplate(ctx context.Context, id string) (*api.AvailabilityTemplateResponse, error) {
	const op = "service.GetAvailabilityTemplate"

	template, err := s.store.GetAvailabilityTemplate(ctx, id)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	startTime := template.RecurrenceStartTime
	endTime :=  template.RecurrenceEndTime

	return &api.AvailabilityTemplateResponse{
		ID:                  template.ID,
		TeacherID:           template.TeacherID,
		Recurrence: api.RecurrenceConfig{
			Days:      template.RecurrenceDays,
			StartTime: startTime.Format("15:04"),
			EndTime:   endTime.Format("15:04"),
		},
		SlotDurationMinutes: template.SlotDurationMinutes,
		StartDate:           template.StartDate.Format("2006-01-02"),
		EndDate:             template.EndDate.Format("2006-01-02"),
		Enabled:             template.Enabled,
	}, nil
}

func (s *Service) UpdateAvailabilityTemplate(ctx context.Context, id string, req *api.AvailabilityTemplateRequest) (*api.AvailabilityTemplateResponse, error) {
	const op = "service.UpdateAvailabilityTemplate"

	template, err := s.store.GetAvailabilityTemplate(ctx, id)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid start_date: %w", op, err)
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid end_date: %w", op, err)
	}

	startTime, err := time.Parse("15:04", req.Recurrence.StartTime)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid start_time: %w", op, err)
	}

	endTime, err := time.Parse("15:04", req.Recurrence.EndTime)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid end_time: %w", op, err)
	}

	template.TeacherID = req.TeacherID
	template.RecurrenceDays = req.Recurrence.Days
	template.RecurrenceStartTime = startTime
	template.RecurrenceEndTime = endTime
	template.SlotDurationMinutes = req.SlotDurationMinutes
	template.StartDate = startDate
	template.EndDate = endDate
	template.Enabled = req.Enabled

	err = s.store.UpdateAvailabilityTemplate(ctx, template)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return s.GetAvailabilityTemplate(ctx, id)
}

func (s *Service) DeleteAvailabilityTemplate(ctx context.Context, id string) error {
	const op = "service.DeleteAvailabilityTemplate"

	err := s.store.DeleteAvailabilityTemplate(ctx, id)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			return fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// Time Blocks

func (s *Service) CreateTimeBlock(ctx context.Context, req *api.TimeBlockRequest) (*api.TimeBlockResponse, error) {
	const op = "service.CreateTimeBlock"

	start, err := time.Parse(time.RFC3339, req.Start)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid start: %w", op, err)
	}

	end, err := time.Parse(time.RFC3339, req.End)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid end: %w", op, err)
	}

	blockType := models.TimeBlockType(req.Type)
	if blockType != models.TimeBlockVacation && blockType != models.TimeBlockSick && blockType != models.TimeBlockOther {
		return nil, fmt.Errorf("%s: invalid type", op)
	}

	block := &models.TimeBlock{
		TeacherID: req.TeacherID,
		Start:     start,
		End:       end,
		Reason:    req.Reason,
		Type:      blockType,
	}

	id, err := s.store.CreateTimeBlock(ctx, block)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return s.GetTimeBlock(ctx, id)
}

func (s *Service) GetTimeBlock(ctx context.Context, id string) (*api.TimeBlockResponse, error) {
	const op = "service.GetTimeBlock"

	block, err := s.store.GetTimeBlock(ctx, id)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &api.TimeBlockResponse{
		ID:        block.ID,
		TeacherID: block.TeacherID,
		Start:     block.Start,
		End:       block.End,
		Reason:    block.Reason,
		Type:      string(block.Type),
	}, nil
}

func (s *Service) ListTimeBlocks(ctx context.Context, teacherID *string, from, to *time.Time) ([]*api.TimeBlockResponse, error) {
	const op = "service.ListTimeBlocks"

	blocks, err := s.store.ListTimeBlocks(ctx, teacherID, from, to)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	result := make([]*api.TimeBlockResponse, 0, len(blocks))
	for _, block := range blocks {
		result = append(result, &api.TimeBlockResponse{
			ID:        block.ID,
			TeacherID: block.TeacherID,
			Start:     block.Start,
			End:       block.End,
			Reason:    block.Reason,
			Type:      string(block.Type),
		})
	}

	return result, nil
}

func (s *Service) UpdateTimeBlock(ctx context.Context, id string, req *api.TimeBlockRequest) (*api.TimeBlockResponse, error) {
	const op = "service.UpdateTimeBlock"

	block, err := s.store.GetTimeBlock(ctx, id)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	start, err := time.Parse(time.RFC3339, req.Start)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid start: %w", op, err)
	}

	end, err := time.Parse(time.RFC3339, req.End)
	if err != nil {
		return nil, fmt.Errorf("%s: invalid end: %w", op, err)
	}

	blockType := models.TimeBlockType(req.Type)
	if blockType != models.TimeBlockVacation && blockType != models.TimeBlockSick && blockType != models.TimeBlockOther {
		return nil, fmt.Errorf("%s: invalid type", op)
	}

	block.TeacherID = req.TeacherID
	block.Start = start
	block.End = end
	block.Reason = req.Reason
	block.Type = blockType

	err = s.store.UpdateTimeBlock(ctx, block)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return s.GetTimeBlock(ctx, id)
}

func (s *Service) DeleteTimeBlock(ctx context.Context, id string) error {
	const op = "service.DeleteTimeBlock"

	err := s.store.DeleteTimeBlock(ctx, id)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			return fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// Slots

func (s *Service) GetSlot(ctx context.Context, id string) (*api.SlotResponse, error) {
	const op = "service.GetSlot"

	slot, err := s.store.GetSlot(ctx, id)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &api.SlotResponse{
		ID:         slot.ID,
		Start:      slot.Start,
		End:        slot.End,
		TeacherID:  slot.TeacherID,
		Status:     string(slot.Status),
		BookingID:  slot.BookingID,
		TemplateID: slot.TemplateID,
	}, nil
}

func (s *Service) ListSlots(ctx context.Context, filters *SlotFilters) ([]*api.SlotResponse, error) {
	const op = "service.ListSlots"

	// Convert service.SlotFilters to interface{} for storage layer
	// Storage will handle the conversion
	slots, err := s.store.ListSlots(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	result := make([]*api.SlotResponse, 0, len(slots))
	for _, slot := range slots {
		result = append(result, &api.SlotResponse{
			ID:         slot.ID,
			Start:      slot.Start,
			End:        slot.End,
			TeacherID:  slot.TeacherID,
			Status:     string(slot.Status),
			BookingID:  slot.BookingID,
			TemplateID: slot.TemplateID,
		})
	}

	return result, nil
}

func (s *Service) GetSlotsByIDs(ctx context.Context, ids []string) ([]*api.SlotResponse, error) {
	const op = "service.GetSlotsByIDs"

	slots, err := s.store.GetSlotsByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	result := make([]*api.SlotResponse, 0, len(slots))
	for _, slot := range slots {
		result = append(result, &api.SlotResponse{
			ID:         slot.ID,
			Start:      slot.Start,
			End:        slot.End,
			TeacherID:  slot.TeacherID,
			Status:     string(slot.Status),
			BookingID:  slot.BookingID,
			TemplateID: slot.TemplateID,
		})
	}

	return result, nil
}

// пример — требуется адаптация под вашу реализацию store/tx types
func (s *Service) GenerateSlots(ctx context.Context, req *api.SlotGenerateRequest) (string, error) {
	const op = "service.GenerateSlots"

	from, err := time.Parse(time.RFC3339, req.From)
	if err != nil {
		return "", fmt.Errorf("%s: invalid from: %w", op, err)
	}
	to, err := time.Parse(time.RFC3339, req.To)
	if err != nil {
		return "", fmt.Errorf("%s: invalid to: %w", op, err)
	}
	if to.Before(from) {
		return "", fmt.Errorf("%s: to is before from", op)
	}

	tpl, err := s.store.GetAvailabilityTemplate(ctx, *req.TemplateID)
	if err != nil {
		return "", fmt.Errorf("%s: get template: %w", op, err)
	}
	if !tpl.Enabled {
		return "", fmt.Errorf("%s: template disabled", op)
	}

	// пересечение диапазонов: используем date-поляну шаблона
	// tpl.StartDate / tpl.EndDate — это DATE, полагаем, что время ноль.
	// делаем genFrom/genTo в том же location, что и "from"
	loc := from.Location()
	// нормализуем tpl.StartDate/EndDate к midnight в loc
	tplStart := time.Date(tpl.StartDate.Year(), tpl.StartDate.Month(), tpl.StartDate.Day(), 0, 0, 0, 0, loc)
	tplEnd := time.Date(tpl.EndDate.Year(), tpl.EndDate.Month(), tpl.EndDate.Day(), 23, 59, 59, 0, loc)

	genFrom := from
	if genFrom.Before(tplStart) {
		genFrom = tplStart
	}
	genTo := to
	if genTo.After(tplEnd) {
		genTo = tplEnd
	}
	if genFrom.After(genTo) {
		return "", fmt.Errorf("%s: no dates to generate after intersecting template bounds", op)
	}

	// подготовим карту допустимых дней недели
	allowed := map[int]struct{}{}
	for _, d := range tpl.RecurrenceDays {
		if wd, ok := parseWeekdayFlexible(d); ok {
			allowed[int(wd)] = struct{}{}
		}
	}

	
	// получаем часы/минуты из TIME полей
	startH := tpl.RecurrenceStartTime.Hour()
	startM := tpl.RecurrenceStartTime.Minute()
	endH := tpl.RecurrenceEndTime.Hour()
	endM := tpl.RecurrenceEndTime.Minute()

	// duration
	slotDur := time.Duration(tpl.SlotDurationMinutes) * time.Minute
	if slotDur <= 0 {
		return "", fmt.Errorf("%s: invalid slot duration: %d", op, tpl.SlotDurationMinutes)
	}

	// начинаем транзакцию и гарантированный откат, если не закоммитим
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return "", fmt.Errorf("%s: begin tx: %w", op, err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// перебор по дням (genFrom..genTo) включительно
	for d := truncateToDate(genFrom, loc); !d.After(truncateToDate(genTo, loc)); d = d.AddDate(0, 0, 1) {
		wd := int(d.Weekday()) // 0=Sunday
		if _, ok := allowed[wd]; !ok {
			continue
		}

		dayStart := time.Date(d.Year(), d.Month(), d.Day(), startH, startM, 0, 0, loc)
		dayEnd := time.Date(d.Year(), d.Month(), d.Day(), endH, endM, 0, 0, loc)

		// если конец <= старт -> skip
		if !dayEnd.After(dayStart) {
			continue
		}

		// генерируем слоты: условие cur + slotDur <= dayEnd
		for cur := dayStart; !cur.Add(slotDur).After(dayEnd); cur = cur.Add(slotDur) {
			slot := &models.Slot{
				TeacherID:  tpl.TeacherID,
				Start:      cur,
				End:        cur.Add(slotDur),
				Status:     models.SlotFree,
				TemplateID: &tpl.ID,
			}
			if _, err := s.store.CreateSlot(ctx, tx, slot); err != nil {
				return "", fmt.Errorf("%s: create slot: %w", op, err)
			}
		}
	}

	// коммит
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("%s: commit: %w", op, err)
	}

	jobID := fmt.Sprintf("job-%d", time.Now().Unix())
	return jobID, nil
}

// truncateToDate возвращает дату с нулевым временем в указанной локации
func truncateToDate(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

// parseWeekdayFlexible поддерживает форматы, которые часто лежат в TEXT[]:
// "mon","monday","Mon","1","0" и т.д. (0 = Sunday)
func parseWeekdayFlexible(s string) (time.Weekday, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, false
	}
	// numeric
	if n, err := strconv.Atoi(s); err == nil {
		// допустим 0..6 (Sunday=0)
		if n >= 0 && n <= 6 {
			return time.Weekday(n), true
		}
		// допустим 1..7 (Mon=1..Sun=7)
		if n >= 1 && n <= 7 {
			if n == 7 {
				return time.Sunday, true
			}
			return time.Weekday(n), true
		}
	}

	switch s {
	case "sun", "sunday":
		return time.Sunday, true
	case "mon", "monday":
		return time.Monday, true
	case "tue", "tues", "tuesday":
		return time.Tuesday, true
	case "wed", "wednesday":
		return time.Wednesday, true
	case "thu", "thur", "thursday":
		return time.Thursday, true
	case "fri", "friday":
		return time.Friday, true
	case "sat", "saturday":
		return time.Saturday, true
	default:
		return 0, false
	}
}



// Bookings

func (s *Service) CreateBooking(ctx context.Context, req *api.BookingRequest, idempotencyKey *string) (*api.BookingResponse, error) {
	const op = "service.CreateBooking"

	lockKey := fmt.Sprintf("slot:%s", req.SlotID)
	
	locked, err := s.locker.Lock(ctx, lockKey, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("%s: lock error: %w", op, err)
	}
	if !locked {
		return nil, fmt.Errorf("%s: %w", op, response.ErrLocked)
	}
	defer func() {
		_ = s.locker.Unlock(ctx, lockKey)
	}()

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: begin tx: %w", op, err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	slot, err := s.store.GetSlotForBooking(ctx, req.SlotID)
	if err != nil {
		_ = tx.Rollback()
    	if errors.Is(err, response.ErrNotFound) {
        	return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}
		if errors.Is(err, response.ErrSlotNotAvailable) {
        	return nil, fmt.Errorf("%s: %w", op, response.ErrSlotNotAvailable)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	booking := &models.Booking{
		SlotID:    req.SlotID,
		StudentID: req.StudentID,
		TeacherID: slot.TeacherID,
		Status:    models.BookingPending,
	}

	bookingID, err := s.store.CreateBooking(ctx, tx, booking)
    if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("%s: create booking: %w", op, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("%s: commit: %w", op, err)
	}

	return s.GetBooking(ctx, bookingID)
}

func (s *Service) GetBooking(ctx context.Context, id string) (*api.BookingResponse, error) {
	const op = "service.GetBooking"

	booking, err := s.store.GetBooking(ctx, id)
    if err != nil {
        if errors.Is(err, response.ErrNotFound) {
            return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
        }
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &api.BookingResponse{
		ID:        booking.ID,
		SlotID:    booking.SlotID,
		StudentID: booking.StudentID,
		TeacherID: booking.TeacherID,
		Status:    string(booking.Status),
	}, nil
}

func (s *Service) ListBookings(ctx context.Context, studentID, teacherID *string, from, to *time.Time, status *string) ([]*api.BookingResponse, error) {
	const op = "service.ListBookings"

	bookings, err := s.store.ListBookings(ctx, studentID, teacherID, from, to, status)
	if err != nil {
        return nil, fmt.Errorf("%s: %w", op, err)
    }

	result := make([]*api.BookingResponse, 0, len(bookings))
	for _, booking := range bookings {
		result = append(result, &api.BookingResponse{
			ID:        booking.ID,
			SlotID:    booking.SlotID,
			StudentID: booking.StudentID,
			TeacherID: booking.TeacherID,
			Status:    string(booking.Status),
		})
	}

	return result, nil
}

func (s *Service) CancelBooking(ctx context.Context, bookingID string) (*api.BookingResponse, error) {
	const op = "service.CancelBooking"

	booking, err := s.store.GetBooking(ctx, bookingID)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

    tx, err := s.store.BeginTx(ctx)
	if err != nil {
        return nil, fmt.Errorf("%s: begin tx: %w", op, err)
    }

	defer func() {
		if p := recover(); p != nil {
            _ = tx.Rollback()
            panic(p)
		}
	}()

	err = s.store.UpdateBookingStatus(ctx, bookingID, models.BookingCancelled)
	if err != nil {
            _ = tx.Rollback()
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	// Free the slot
	err = s.store.UpdateSlotStatus(ctx, booking.SlotID, models.SlotFree, nil)
	if err != nil {
        _ = tx.Rollback()
		return nil, fmt.Errorf("%s: %w", op, err)
    }

    if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("%s: commit: %w", op, err)
	}

	return s.GetBooking(ctx, bookingID)
}

func (s *Service) ConfirmBooking(ctx context.Context, bookingID string) (*api.BookingResponse, error) {
	const op = "service.ConfirmBooking"

	err := s.store.UpdateBookingStatus(ctx, bookingID, models.BookingConfirmed)
    if err != nil {
        if errors.Is(err, response.ErrNotFound) {
            return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
        }
        return nil, fmt.Errorf("%s: %w", op, err)
    }

	return s.GetBooking(ctx, bookingID)
}

func (s *Service) RescheduleBooking(ctx context.Context, bookingID string, newSlotId string) (*api.BookingResponse, error) {
	const op = "service.RescheduleBooking"

	_, err := s.store.GetBooking(ctx, bookingID)
    if err != nil {
        if errors.Is(err, response.ErrNotFound) {
            return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
        }
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	newSlot, err := s.store.GetSlot(ctx, newSlotId)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			return nil, fmt.Errorf("%s: new slot not found: %w", op, response.ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if newSlot.Status != models.SlotFree {
		return nil, fmt.Errorf("%s: %w", op, response.ErrSlotNotAvailable)
	}

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: begin tx: %w", op, err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	err = s.store.RescheduleBooking(ctx, tx, bookingID, newSlotId)
	if err != nil {
		_ = tx.Rollback()
        return nil, fmt.Errorf("%s: %w", op, err)
    }

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("%s: commit: %w", op, err)
	}

	return s.GetBooking(ctx, bookingID)
}

func (s *Service) DeleteBooking(ctx context.Context, bookingID string) error {
	const op = "service.DeleteBooking"

	booking, err := s.store.GetBooking(ctx, bookingID)
	if err != nil {
		if errors.Is(err, response.ErrNotFound) {
			return fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("%s: begin tx: %w", op, err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	err = s.store.DeleteBooking(ctx, bookingID)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("%s: %w", op, err)
	}

	// Free the slot
	err = s.store.UpdateSlotStatus(ctx, booking.SlotID, models.SlotFree, nil)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("%s: %w", op, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: commit: %w", op, err)
	}

	return nil
}

// Attendance

func (s *Service) CreateAttendance(ctx context.Context, req *api.AttendanceRequest) (*api.AttendanceResponse, error) {
	const op = "service.CreateAttendance"

	status := models.AttendanceStatus(req.Status)
	if status != models.AttendancePresent && status != models.AttendanceAbsent && status != models.AttendanceLate {
		return nil, fmt.Errorf("%s: invalid status", op)
	}

	attendance := &models.Attendance{
		BookingID: req.BookingID,
		Status:    status,
		Notes:     req.Notes,
	}

	id, err := s.store.CreateAttendance(ctx, attendance)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return s.GetAttendance(ctx, id)
}

func (s *Service) GetAttendance(ctx context.Context, id string) (*api.AttendanceResponse, error) {
	const op = "service.GetAttendance"

	attendance, err := s.store.GetAttendance(ctx, id)
    if err != nil {
        if errors.Is(err, response.ErrNotFound) {
            return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
        }
        return nil, fmt.Errorf("%s: %w", op, err)
    }

	return &api.AttendanceResponse{
		ID:        attendance.ID,
		BookingID: attendance.BookingID,
		Status:    string(attendance.Status),
		Notes:     attendance.Notes,
	}, nil
}

func (s *Service) ListAttendance(ctx context.Context, teacherID *string, from, to *time.Time) ([]*api.AttendanceResponse, error) {
	const op = "service.ListAttendance"

	attendances, err := s.store.ListAttendance(ctx, teacherID, from, to)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	result := make([]*api.AttendanceResponse, 0, len(attendances))
	for _, attendance := range attendances {
		result = append(result, &api.AttendanceResponse{
			ID:        attendance.ID,
			BookingID: attendance.BookingID,
			Status:    string(attendance.Status),
			Notes:     attendance.Notes,
		})
	}

	return result, nil
}
