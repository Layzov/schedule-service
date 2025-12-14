package models

import "time"

type AvailabilityTemplate struct {
	ID                  string    `db:"id"`
	TeacherID           string    `db:"teacher_id"`
	RecurrenceDays      []string  `db:"recurrence_days"`
	RecurrenceStartTime string    `db:"recurrence_start_time"`
	RecurrenceEndTime   string    `db:"recurrence_end_time"`
	SlotDurationMinutes int       `db:"slot_duration_minutes"`
	StartDate           time.Time `db:"start_date"`
	EndDate             time.Time `db:"end_date"`
	Enabled             bool      `db:"enabled"`
}

type AvailabilityTplSlot struct {
	ID                  string    `db:"id"`
	TeacherID           string    `db:"teacher_id"`
	RecurrenceDays      []string  `db:"recurrence_days"`
	RecurrenceStartTime time.Time    `db:"recurrence_start_time"`
	RecurrenceEndTime   time.Time    `db:"recurrence_end_time"`
	SlotDurationMinutes int       `db:"slot_duration_minutes"`
	StartDate           time.Time `db:"start_date"`
	EndDate             time.Time `db:"end_date"`
	Enabled             bool      `db:"enabled"`
}


type TimeBlockType string

const (
	TimeBlockVacation TimeBlockType = "vacation"
	TimeBlockSick     TimeBlockType = "sick"
	TimeBlockOther    TimeBlockType = "other"
)

type TimeBlock struct {
	ID        string         `db:"id"`
	TeacherID string         `db:"teacher_id"`
	Start     time.Time      `db:"start"`
	End       time.Time      `db:"end"`
	Reason    string         `db:"reason"`
	Type      TimeBlockType  `db:"type"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
}

type SlotStatus string

const (
	SlotFree      SlotStatus = "free"
	SlotBooked    SlotStatus = "booked"
	SlotCancelled SlotStatus = "cancelled"
	SlotBlocked   SlotStatus = "blocked"
)

type Slot struct {
	ID         string     `db:"id"`
	TeacherID  string     `db:"teacher_id"`
	Start      time.Time  `db:"start"`
	End        time.Time  `db:"end"`
	Status     SlotStatus `db:"status"`
	BookingID  *string    `db:"booking_id"`
	TemplateID *string    `db:"template_id"`
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at"`
}

type BookingStatus string

const (
	BookingPending  BookingStatus = "pending"
	BookingConfirmed BookingStatus = "confirmed"
	BookingCancelled BookingStatus = "cancelled"
)

type Booking struct {
	ID          string        `db:"id"`
	SlotID      string        `db:"slot_id"`
	StudentID   string        `db:"student_id"`
	TeacherID   string        `db:"teacher_id"`
	Status      BookingStatus `db:"status"`
}

type AttendanceStatus string

const (
	AttendancePresent AttendanceStatus = "present"
	AttendanceAbsent  AttendanceStatus = "absent"
	AttendanceLate    AttendanceStatus = "late"
)

type Attendance struct {
	ID        string           `db:"id"`
	BookingID string           `db:"booking_id"`
	Status    AttendanceStatus `db:"status"`
	Notes     string           `db:"notes"`
	CreatedAt time.Time        `db:"created_at"`
	UpdatedAt time.Time        `db:"updated_at"`
}
