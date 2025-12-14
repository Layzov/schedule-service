package api

import "time"

// Availability Templates
type AvailabilityTemplateRequest struct {
	TeacherID           string           `json:"teacher_id"`
	Recurrence          RecurrenceConfig `json:"recurrence"`
	SlotDurationMinutes int              `json:"slot_duration_minutes"`
	StartDate           string           `json:"start_date"`
	EndDate             string           `json:"end_date"`
	Enabled             bool             `json:"enabled"`
}

type RecurrenceConfig struct {
	Days      []string `json:"days"`
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time"`
}

type AvailabilityTemplateResponse struct {
	ID                  string           `json:"id"`
	TeacherID           string           `json:"teacher_id"`
	Recurrence          RecurrenceConfig `json:"recurrence"`
	SlotDurationMinutes int              `json:"slot_duration_minutes"`
	StartDate           string           `json:"start_date"`
	EndDate             string           `json:"end_date"`
	Enabled             bool             `json:"enabled"`
}

// Time Blocks
type TimeBlockRequest struct {
	TeacherID string `json:"teacher_id"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Reason    string `json:"reason"`
	Type      string `json:"type"`
}

type TimeBlockResponse struct {
	ID        string    `json:"id"`
	TeacherID string    `json:"teacher_id"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Reason    string    `json:"reason"`
	Type      string    `json:"type"`
}

// Slots
type SlotResponse struct {
	ID         string     `json:"id"`
	Start      time.Time  `json:"start"`
	End        time.Time  `json:"end"`
	TeacherID  string     `json:"teacher_id"`
	Status     string     `json:"status"`
	BookingID  *string    `json:"booking_id,omitempty"`
	TemplateID *string    `json:"template_id,omitempty"`
}

type SlotGenerateRequest struct {
	TemplateID *string `json:"template_id,omitempty"`
	TeacherID  *string `json:"teacher_id,omitempty"`
	From       string  `json:"from"`
	To         string  `json:"to"`
}

type SlotGenerateResponse struct {
	JobID string `json:"job_id"`
}

type SlotBatchRequest struct {
	IDs []string `json:"ids"`
}

// Bookings
type BookingRequest struct {
	SlotID    string                 `json:"slot_id"`
	StudentID string                 `json:"student_id"`
}

type BookingResponse struct {
	ID        string                 `json:"id"`
	SlotID    string                 `json:"slot_id"`
	StudentID string                 `json:"student_id"`
	TeacherID string                 `json:"teacher_id"`
	Status    string                 `json:"status"`
}

type BookingRescheduleRequest struct {
	BookingID string `json:"booking_id"`
	NewSlotID string `json:"new_slot_id"`
}

// Attendance
type AttendanceRequest struct {
	BookingID string `json:"booking_id"`
	Status    string `json:"status"`
	Notes     string `json:"notes"`
}

type AttendanceResponse struct {
	ID        string    `json:"id"`
	BookingID string    `json:"booking_id"`
	Status    string    `json:"status"`
	Notes     string    `json:"notes"`
}
