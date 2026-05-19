package repository

import (
	"context"
	"database/sql"

	"clinic-appointment/internal/models"
)

type DepartmentRepository interface {
	List(ctx context.Context) ([]models.Department, error)
	GetByID(ctx context.Context, id int64) (*models.Department, error)
}

type DoctorRepository interface {
	ListByDepartment(ctx context.Context, departmentID int64) ([]models.Doctor, error)
	GetByID(ctx context.Context, id int64) (*models.Doctor, error)
}

type ScheduleSlotRepository interface {
	ListByDoctorAndDateRange(ctx context.Context, doctorID int64, startDate, endDate string) ([]models.ScheduleSlot, error)
	GetByID(ctx context.Context, id int64) (*models.ScheduleSlot, error)
	GetByDoctorDateTime(ctx context.Context, doctorID int64, date, startTime, endTime string) (*models.ScheduleSlot, error)
	Create(ctx context.Context, slot *models.ScheduleSlot) error
	UpdateUsedQuota(ctx context.Context, tx *sql.Tx, slotID int64, delta int) error
	MarkSuspended(ctx context.Context, doctorID int64, date string) (int64, error)
	Delete(ctx context.Context, id int64) error
}

type AppointmentRepository interface {
	Create(ctx context.Context, tx *sql.Tx, appt *models.Appointment) error
	GetByID(ctx context.Context, id int64) (*models.Appointment, error)
	GetByNo(ctx context.Context, appointmentNo string) (*models.Appointment, error)
	UpdateStatus(ctx context.Context, tx *sql.Tx, id int64, status models.AppointmentStatus, operator, reason string) error
	ListByPatient(ctx context.Context, patientPhone string, status models.AppointmentStatus, date string, offset, limit int) ([]models.Appointment, int64, error)
	ListBySlotIDAndStatus(ctx context.Context, slotID int64, status models.AppointmentStatus) ([]models.Appointment, error)
	ListByDoctorAndDate(ctx context.Context, doctorID int64, date string, status models.AppointmentStatus, offset, limit int) ([]models.Appointment, int64, error)
	CancelBySlotID(ctx context.Context, tx *sql.Tx, slotID int64, operator, reason string) (int64, error)
}

type AppointmentLogRepository interface {
	Create(ctx context.Context, log *models.AppointmentLog) error
	ListByAppointmentID(ctx context.Context, appointmentID int64) ([]models.AppointmentLog, error)
}

type SuspensionRepository interface {
	Create(ctx context.Context, tx *sql.Tx, suspension *models.SuspensionDay) error
	Exists(ctx context.Context, doctorID int64, date string) (bool, error)
}
