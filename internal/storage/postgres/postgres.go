package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"rasp-service/internal/models"
	"rasp-service/pkg/response"
	"time"

	"github.com/lib/pq"
)

type Storage struct {
	db *sql.DB
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.postgres.New"

	db, err := sql.Open("postgres", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	if s.db == nil || s == nil {
		return nil
	}

	return s.db.Close()
}

func (s *Storage) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return s.db.BeginTx(ctx, nil)
}

// Availability Templates

func (s *Storage) CreateAvailabilityTemplate(ctx context.Context, template *models.AvailabilityTemplate) (string, error) {
	const op = "storage.postgres.CreateAvailabilityTemplate"

	var id string
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO availability_templates 
		(teacher_id, recurrence_days, recurrence_start_time, recurrence_end_time, 
		 slot_duration_minutes, start_date, end_date, enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		template.TeacherID,
		pq.Array(template.RecurrenceDays),
		template.RecurrenceStartTime,
		template.RecurrenceEndTime,
		template.SlotDurationMinutes,
		template.StartDate,
		template.EndDate,
		template.Enabled,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetAvailabilityTemplate(ctx context.Context, id string) (*models.AvailabilityTplSlot, error) {
	const op = "storage.postgres.GetAvailabilityTemplate"

	var template models.AvailabilityTplSlot
	var recurrenceDays pq.StringArray

	err := s.db.QueryRowContext(ctx,
		`SELECT id, teacher_id, recurrence_days, recurrence_start_time, recurrence_end_time,
		 slot_duration_minutes, start_date, end_date, enabled
		 FROM availability_templates WHERE id = $1`,
		id,
	).Scan(
		&template.ID,
		&template.TeacherID,
		&recurrenceDays,
		&template.RecurrenceStartTime,
		&template.RecurrenceEndTime,
		&template.SlotDurationMinutes,
		&template.StartDate,
		&template.EndDate,
		&template.Enabled,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	template.RecurrenceDays = []string(recurrenceDays)

	return &template, nil
}


func (s *Storage) UpdateAvailabilityTemplate(ctx context.Context, template *models.AvailabilityTplSlot) error {
	const op = "storage.postgres.UpdateAvailabilityTemplate"

	res, err := s.db.ExecContext(ctx,
		`UPDATE availability_templates 
		SET teacher_id = $1, recurrence_days = $2, recurrence_start_time = $3, 
		    recurrence_end_time = $4, slot_duration_minutes = $5, start_date = $6,
		    end_date = $7, enabled = $8
		WHERE id = $9`,
		template.TeacherID,
		pq.Array(template.RecurrenceDays),
		template.RecurrenceStartTime,
		template.RecurrenceEndTime,
		template.SlotDurationMinutes,
		template.StartDate,
		template.EndDate,
		template.Enabled,
		template.ID,
	)

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", op, response.ErrNotFound)
	}

	return nil
}

func (s *Storage) DeleteAvailabilityTemplate(ctx context.Context, id string) error {
	const op = "storage.postgres.DeleteAvailabilityTemplate"

	res, err := s.db.ExecContext(ctx, `DELETE FROM availability_templates WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", op, response.ErrNotFound)
	}

	return nil
}

// Time Blocks

func (s *Storage) CreateTimeBlock(ctx context.Context, block *models.TimeBlock) (string, error) {
	const op = "storage.postgres.CreateTimeBlock"

	var id string
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO time_blocks (teacher_id, start, "end", reason, type)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		block.TeacherID,
		block.Start,
		block.End,
		block.Reason,
		string(block.Type),
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetTimeBlock(ctx context.Context, id string) (*models.TimeBlock, error) {
	const op = "storage.postgres.GetTimeBlock"

	var block models.TimeBlock
	var blockType string

	err := s.db.QueryRowContext(ctx,
		`SELECT id, teacher_id, start, "end", reason, type, created_at, updated_at
		 FROM time_blocks WHERE id = $1`,
		id,
	).Scan(
		&block.ID,
		&block.TeacherID,
		&block.Start,
		&block.End,
		&block.Reason,
		&blockType,
		&block.CreatedAt,
		&block.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	block.Type = models.TimeBlockType(blockType)

	return &block, nil
}

func (s *Storage) ListTimeBlocks(ctx context.Context, teacherID *string, from, to *time.Time) ([]*models.TimeBlock, error) {
	const op = "storage.postgres.ListTimeBlocks"

	query := `SELECT id, teacher_id, start, "end", reason, type, created_at, updated_at FROM time_blocks WHERE 1=1`
	args := []interface{}{}
	argPos := 1

	if teacherID != nil {
		query += fmt.Sprintf(" AND teacher_id = $%d", argPos)
		args = append(args, *teacherID)
		argPos++
	}

	if from != nil {
		query += fmt.Sprintf(" AND \"end\" >= $%d", argPos)
		args = append(args, *from)
		argPos++
	}

	if to != nil {
		query += fmt.Sprintf(" AND start <= $%d", argPos)
		args = append(args, *to)
		argPos++
	}

	query += " ORDER BY start DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var blocks []*models.TimeBlock
	for rows.Next() {
		var block models.TimeBlock
		var blockType string

		err := rows.Scan(
			&block.ID,
			&block.TeacherID,
			&block.Start,
			&block.End,
			&block.Reason,
			&blockType,
			&block.CreatedAt,
			&block.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		block.Type = models.TimeBlockType(blockType)
		blocks = append(blocks, &block)
	}

	return blocks, nil
}

func (s *Storage) UpdateTimeBlock(ctx context.Context, block *models.TimeBlock) error {
	const op = "storage.postgres.UpdateTimeBlock"

	res, err := s.db.ExecContext(ctx,
		`UPDATE time_blocks 
		SET teacher_id = $1, start = $2, "end" = $3, reason = $4, type = $5
		WHERE id = $6`,
		block.TeacherID,
		block.Start,
		block.End,
		block.Reason,
		string(block.Type),
		block.ID,
	)

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", op, response.ErrNotFound)
	}

	return nil
}

func (s *Storage) DeleteTimeBlock(ctx context.Context, id string) error {
	const op = "storage.postgres.DeleteTimeBlock"

	res, err := s.db.ExecContext(ctx, `DELETE FROM time_blocks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", op, response.ErrNotFound)
	}

	return nil
}

// Slots

func (s *Storage) GetSlot(ctx context.Context, id string) (*models.Slot, error) {
	const op = "storage.postgres.GetSlot"

	var slot models.Slot
	var status string
	var bookingID, templateID sql.NullString

	err := s.db.QueryRowContext(ctx,
		`SELECT id, teacher_id, starts_at, ends_at, status, booking_id, template_id, created_at, updated_at
		 FROM slots WHERE id = $1`,
		id,
	).Scan(
		&slot.ID,
		&slot.TeacherID,
		&slot.Start,
		&slot.End,
		&status,
		&bookingID,
		&templateID,
		&slot.CreatedAt,
		&slot.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	slot.Status = models.SlotStatus(status)
	if bookingID.Valid {
		slot.BookingID = &bookingID.String
	}
	if templateID.Valid {
		slot.TemplateID = &templateID.String
	}

	return &slot, nil
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

func (s *Storage) ListSlots(ctx context.Context, filters interface{}) ([]*models.Slot, error) {
	const op = "storage.postgres.ListSlots"

	query := `SELECT id, teacher_id, starts_at, ends_at, status, booking_id, template_id, created_at, updated_at FROM slots WHERE 1=1`
	args := []interface{}{}
	argPos := 1

	if filters != nil {
		f, ok := filters.(*SlotFilters)
		if !ok {
			return nil, fmt.Errorf("%s: invalid filters type", op)
		}
		if f.TeacherID != nil {
			query += fmt.Sprintf(" AND teacher_id = $%d", argPos)
			args = append(args, *f.TeacherID)
			argPos++
		}

		if f.From != nil {
			query += fmt.Sprintf(" AND starts_at >= $%d", argPos)
			args = append(args, *f.From)
			argPos++
		}

		if f.To != nil {
			query += fmt.Sprintf(" AND ends_at <= $%d", argPos)
			args = append(args, *f.To)
			argPos++
		}

		if f.Status != nil {
			query += fmt.Sprintf(" AND status = $%d", argPos)
			args = append(args, *f.Status)
			argPos++
		}

		if f.Duration != nil {
			query += fmt.Sprintf(" AND EXTRACT(EPOCH FROM (ends_at - starts_at))/60 = $%d", argPos)
			args = append(args, *f.Duration)
			argPos++
		}

		if f.Q != nil && *f.Q != "" {
			query += fmt.Sprintf(" AND (teacher_id ILIKE $%d OR id::text ILIKE $%d)", argPos, argPos)
			searchPattern := "%" + *f.Q + "%"
			args = append(args, searchPattern, searchPattern)
			argPos += 2
		}

		// Sort - валидация для предотвращения SQL injection
		sortBy := "starts_at"
		if f.Sort != nil && *f.Sort != "" {
			allowedSortFields := map[string]bool{
				"starts_at":  true,
				"ends_at":    true,
				"status":     true,
				"teacher_id": true,
				"created_at": true,
			}
			if allowedSortFields[*f.Sort] {
				sortBy = *f.Sort
			}
		}
		query += fmt.Sprintf(" ORDER BY %s", sortBy)

		// Применяем пагинацию, если указаны page и per_page
		if f.Page != nil && f.PerPage != nil {
			page := *f.Page
			perPage := *f.PerPage
			// Валидация значений
			if page < 1 {
				page = 1
			}
			if perPage < 1 {
				perPage = 20
			}
			if perPage > 100 {
				perPage = 100
			}
			offset := (page - 1) * perPage
			query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1)
			args = append(args, perPage, offset)
		}
	} else {
		// Если фильтры не указаны, применяем дефолтные значения
		query += " ORDER BY starts_at LIMIT 20 OFFSET 0"
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var slots []*models.Slot
	for rows.Next() {
		var slot models.Slot
		var status string
		var bookingID, templateID sql.NullString

		err := rows.Scan(
			&slot.ID,
			&slot.TeacherID,
			&slot.Start,
			&slot.End,
			&status,
			&bookingID,
			&templateID,
			&slot.CreatedAt,
			&slot.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		slot.Status = models.SlotStatus(status)
		if bookingID.Valid {
			slot.BookingID = &bookingID.String
		}
		if templateID.Valid {
			slot.TemplateID = &templateID.String
		}

		slots = append(slots, &slot)
	}

	return slots, nil
}

func (s *Storage) GetSlotsByIDs(ctx context.Context, ids []string) ([]*models.Slot, error) {
	const op = "storage.postgres.GetSlotsByIDs"

	if len(ids) == 0 {
		return []*models.Slot{}, nil
	}

	query := `SELECT id, teacher_id, starts_at, ends_at, status, booking_id, template_id, created_at, updated_at 
			  FROM slots WHERE id = ANY($1)`
	rows, err := s.db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var slots []*models.Slot
	for rows.Next() {
		var slot models.Slot
		var status string
		var bookingID, templateID sql.NullString

		err := rows.Scan(
			&slot.ID,
			&slot.TeacherID,
			&slot.Start,
			&slot.End,
			&status,
			&bookingID,
			&templateID,
			&slot.CreatedAt,
			&slot.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		slot.Status = models.SlotStatus(status)
		if bookingID.Valid {
			slot.BookingID = &bookingID.String
		}
		if templateID.Valid {
			slot.TemplateID = &templateID.String
		}

		slots = append(slots, &slot)
	}

	return slots, nil
}

func (s *Storage) CreateSlot(ctx context.Context, tx *sql.Tx, slot *models.Slot) (string, error) {
	const op = "storage.postgres.CreateSlot"

	var id string
	err := tx.QueryRowContext(ctx,
		`INSERT INTO slots (teacher_id, starts_at, ends_at, status, template_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		slot.TeacherID,
		slot.Start,
		slot.End,
		string(slot.Status),
		slot.TemplateID,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (s *Storage) UpdateSlotStatus(ctx context.Context, slotID string, status models.SlotStatus, bookingID *string) error {
	const op = "storage.postgres.UpdateSlotStatus"

	_, err := s.db.ExecContext(ctx,
		`UPDATE slots SET status = $1, booking_id = $2 WHERE id = $3`,
		string(status),
		bookingID,
		slotID,
	)

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) GetSlotForBooking(ctx context.Context, slotID string) (*models.Slot, error) {
	const op = "storage.postgres.GetSlotForBooking"

	var slot models.Slot
	var status string
	var bookingID, templateID sql.NullString

	err := s.db.QueryRowContext(ctx,
		`SELECT id, teacher_id, starts_at, ends_at, status, booking_id, template_id
		 FROM slots WHERE id = $1 FOR UPDATE`,
		slotID,
	).Scan(
		&slot.ID,
		&slot.TeacherID,
		&slot.Start,
		&slot.End,
		&status,
		&bookingID,
		&templateID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if status != string(models.SlotFree){
		return nil, fmt.Errorf("%s: %w", op, response.ErrSlotNotAvailable)
	}

	return &slot, nil
}


func (s *Storage) CreateBooking(ctx context.Context, tx *sql.Tx, booking *models.Booking) (string, error) {
	const op = "storage.postgres.CreateBooking"

	var id string
	err := tx.QueryRowContext(ctx,
		`INSERT INTO bookings (slot_id, student_id, teacher_id, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		booking.SlotID,
		booking.StudentID,
		booking.TeacherID,
		string(booking.Status),
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	_, err = tx.ExecContext(ctx, 
		`UPDATE slots SET status = $1, booking_id = $2 WHERE id = $3`,
		string(models.SlotBooked), id, booking.SlotID,
	)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetBooking(ctx context.Context, id string) (*models.Booking, error) {
	const op = "storage.postgres.GetBooking"

	var booking models.Booking
	var status string

	err := s.db.QueryRowContext(ctx,
		`SELECT id, slot_id, student_id, teacher_id, status
		 FROM bookings WHERE id = $1`,
		id,
	).Scan(
		&booking.ID,
		&booking.SlotID,
		&booking.StudentID,
		&booking.TeacherID,
		&status,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	booking.Status = models.BookingStatus(status)

	return &booking, nil
}

func (s *Storage) ListBookings(ctx context.Context, studentID, teacherID *string, from, to *time.Time, status *string) ([]*models.Booking, error) {
	const op = "storage.postgres.ListBookings"

	query := `SELECT id, slot_id, student_id, teacher_id, status,
				FROM bookings WHERE 1=1`
	args := []interface{}{}
	argPos := 1

	if studentID != nil {
		query += fmt.Sprintf(" AND student_id = $%d", argPos)
		args = append(args, *studentID)
		argPos++
	}

	if teacherID != nil {
		query += fmt.Sprintf(" AND teacher_id = $%d", argPos)
		args = append(args, *teacherID)
		argPos++
	}

	if from != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argPos)
		args = append(args, *from)
		argPos++
	}

	if to != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argPos)
		args = append(args, *to)
		argPos++
	}

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, *status)
		argPos++
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var bookings []*models.Booking
	for rows.Next() {
		var booking models.Booking
		var status string
		var paymentStr string

		err := rows.Scan(
			&booking.ID,
			&booking.SlotID,
			&booking.StudentID,
			&booking.TeacherID,
			&status,
			&paymentStr,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		booking.Status = models.BookingStatus(status)

		bookings = append(bookings, &booking)
	}

	return bookings, nil
}

func (s *Storage) UpdateBookingStatus(ctx context.Context, bookingID string, status models.BookingStatus) error {
	const op = "storage.postgres.UpdateBookingStatus"

	now := time.Now()
	res, err := s.db.ExecContext(ctx,
		`UPDATE bookings 
		SET status = $1, cancelled_at = CASE WHEN $1 = 'cancelled' THEN $2 ELSE cancelled_at END
		WHERE id = $3`,
		string(status),
		now,
		bookingID,
	)

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", op, response.ErrNotFound)
	}

	return nil
}

func (s *Storage) RescheduleBooking(ctx context.Context, tx *sql.Tx, bookingID, newSlotID string) error {
	const op = "storage.postgres.RescheduleBooking"

	// Get old slot and new slot
	var oldSlotID string
	err := tx.QueryRowContext(ctx, `SELECT slot_id FROM bookings WHERE id = $1`, bookingID).Scan(&oldSlotID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	// Update booking slot
	_, err = tx.ExecContext(ctx, `UPDATE bookings SET slot_id = $1 WHERE id = $2`, newSlotID, bookingID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// Update old slot to free
	_, err = tx.ExecContext(ctx, `UPDATE slots SET status = 'free', booking_id = NULL WHERE id = $1`, oldSlotID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	// Update new slot to booked
	_, err = tx.ExecContext(ctx, `UPDATE slots SET status = 'booked', booking_id = $1 WHERE id = $2`, bookingID, newSlotID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) DeleteBooking(ctx context.Context, bookingID string) error {
	const op = "storage.postgres.DeleteBooking"

	res, err := s.db.ExecContext(ctx, `DELETE FROM bookings WHERE id = $1`, bookingID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", op, response.ErrNotFound)
	}

	return nil
}

// Attendance

func (s *Storage) CreateAttendance(ctx context.Context, attendance *models.Attendance) (string, error) {
	const op = "storage.postgres.CreateAttendance"

	var id string
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO attendance (booking_id, status, notes)
		VALUES ($1, $2, $3)
		RETURNING id`,
		attendance.BookingID,
		string(attendance.Status),
		attendance.Notes,
	).Scan(&id)

	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetAttendance(ctx context.Context, id string) (*models.Attendance, error) {
	const op = "storage.postgres.GetAttendance"

	var attendance models.Attendance
	var status string

	err := s.db.QueryRowContext(ctx,
		`SELECT id, booking_id, status, notes, created_at, updated_at
		 FROM attendance WHERE id = $1`,
		id,
	).Scan(
		&attendance.ID,
		&attendance.BookingID,
		&status,
		&attendance.Notes,
		&attendance.CreatedAt,
		&attendance.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%s: %w", op, response.ErrNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	attendance.Status = models.AttendanceStatus(status)

	return &attendance, nil
}

func (s *Storage) ListAttendance(ctx context.Context, teacherID *string, from, to *time.Time) ([]*models.Attendance, error) {
	const op = "storage.postgres.ListAttendance"

	query := `SELECT a.id, a.booking_id, a.status, a.notes, a.created_at, a.updated_at
			  FROM attendance a
			  JOIN bookings b ON a.booking_id = b.id
			  WHERE 1=1`
	args := []interface{}{}
	argPos := 1

	if teacherID != nil {
		query += fmt.Sprintf(" AND b.teacher_id = $%d", argPos)
		args = append(args, *teacherID)
		argPos++
	}

	if from != nil {
		query += fmt.Sprintf(" AND a.created_at >= $%d", argPos)
		args = append(args, *from)
		argPos++
	}

	if to != nil {
		query += fmt.Sprintf(" AND a.created_at <= $%d", argPos)
		args = append(args, *to)
		argPos++
	}

	query += " ORDER BY a.created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var attendances []*models.Attendance
	for rows.Next() {
		var attendance models.Attendance
		var status string

		err := rows.Scan(
			&attendance.ID,
			&attendance.BookingID,
			&status,
			&attendance.Notes,
			&attendance.CreatedAt,
			&attendance.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		attendance.Status = models.AttendanceStatus(status)
		attendances = append(attendances, &attendance)
	}

	return attendances, nil
}
