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
