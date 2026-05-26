package models

import "time"

type AppointmentStatus string

const (
	StatusPending   AppointmentStatus = "pending"
	StatusConfirmed AppointmentStatus = "confirmed"
	StatusCompleted AppointmentStatus = "completed"
	StatusCancelled AppointmentStatus = "cancelled"
	StatusExpired   AppointmentStatus = "expired"
	StatusSuspended AppointmentStatus = "suspended"
)

type WaitlistStatus string

const (
	WaitlistStatusWaiting   WaitlistStatus = "waiting"
	WaitlistStatusPromoted WaitlistStatus = "promoted"
	WaitlistStatusCancelled WaitlistStatus = "cancelled"
	WaitlistStatusExpired  WaitlistStatus = "expired"
)

type NotificationType string

const (
	NotificationWaitlistPromoted   NotificationType = "waitlist_promoted"
	NotificationApptCancelled  NotificationType = "appointment_cancelled"
	NotificationApptConfirmed NotificationType = "appointment_confirmed"
	NotificationApptSuspended NotificationType = "appointment_suspended"
)

type ConflictType string

const (
	ConflictTypeDuplicateSlot ConflictType = "duplicate_slot"
	ConflictTypeExistingAppt  ConflictType = "existing_appointment"
)

type ScheduleConflict struct {
	Type         ConflictType `json:"type"`
	Date         string       `json:"date"`
	StartTime    string       `json:"start_time"`
	EndTime      string       `json:"end_time"`
	Message      string       `json:"message"`
	ApptCount    int          `json:"appointment_count,omitempty"`
	ExistingSlot *ScheduleSlot `json:"existing_slot,omitempty"`
}

type ScheduleConflictResponse struct {
	HasConflict bool               `json:"has_conflict"`
	Conflicts   []ScheduleConflict `json:"conflicts"`
	Message     string             `json:"message,omitempty"`
}

type Department struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Doctor struct {
	ID            int64     `json:"id"`
	DepartmentID  int64     `json:"department_id"`
	Name          string    `json:"name"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Department    *Department `json:"department,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ScheduleTemplate struct {
	ID         int64     `json:"id"`
	DoctorID   int64     `json:"doctor_id"`
	DayOfWeek  string    `json:"day_of_week"`
	StartTime  string    `json:"start_time"`
	EndTime    string    `json:"end_time"`
	Quota      int       `json:"quota"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ScheduleSlot struct {
	ID            int64     `json:"id"`
	DoctorID      int64     `json:"doctor_id"`
	ScheduleDate  string    `json:"schedule_date"`
	StartTime     string    `json:"start_time"`
	EndTime       string    `json:"end_time"`
	TotalQuota    int       `json:"total_quota"`
	UsedQuota     int       `json:"used_quota"`
	IsSuspended   bool      `json:"is_suspended"`
	Doctor        *Doctor   `json:"doctor,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Appointment struct {
	ID              int64             `json:"id"`
	SlotID          int64             `json:"slot_id"`
	PatientName     string            `json:"patient_name"`
	PatientPhone    string            `json:"patient_phone"`
	PatientIDCard   string            `json:"patient_id_card,omitempty"`
	Status          AppointmentStatus `json:"status"`
	AppointmentNo   string            `json:"appointment_no"`
	Slot            *ScheduleSlot     `json:"slot,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type AppointmentLog struct {
	ID             int64             `json:"id"`
	AppointmentID  int64             `json:"appointment_id"`
	OldStatus      AppointmentStatus `json:"old_status,omitempty"`
	NewStatus      AppointmentStatus `json:"new_status"`
	Operator       string            `json:"operator"`
	Reason         string            `json:"reason,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
}

type SuspensionDay struct {
	ID          int64     `json:"id"`
	DoctorID    int64     `json:"doctor_id"`
	SuspendDate string    `json:"suspend_date"`
	Reason      string    `json:"reason,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Waitlist struct {
	ID             int64           `json:"id"`
	SlotID         int64           `json:"slot_id"`
	PatientName    string          `json:"patient_name"`
	PatientPhone   string          `json:"patient_phone"`
	PatientIDCard  string          `json:"patient_id_card,omitempty"`
	Status         WaitlistStatus  `json:"status"`
	Position       int             `json:"position"`
	AppointmentID  *int64          `json:"appointment_id,omitempty"`
	Slot           *ScheduleSlot   `json:"slot,omitempty"`
	Appointment    *Appointment    `json:"appointment,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type Notification struct {
	ID              int64            `json:"id"`
	Type            NotificationType `json:"type"`
	RecipientPhone  string           `json:"recipient_phone"`
	RecipientName   string           `json:"recipient_name"`
	Content         string           `json:"content"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	IsProcessed     bool             `json:"is_processed"`
	CreatedAt       time.Time        `json:"created_at"`
	ProcessedAt     *time.Time       `json:"processed_at,omitempty"`
}
