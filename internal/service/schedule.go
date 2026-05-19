package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"clinic-appointment/internal/config"
	"clinic-appointment/internal/db"
	"clinic-appointment/internal/logger"
	"clinic-appointment/internal/models"
	"clinic-appointment/internal/repository"

	"go.uber.org/zap"
)

var (
	ErrSlotNotFound       = errors.New("schedule slot not found")
	ErrSlotSuspended      = errors.New("schedule slot is suspended")
	ErrSlotNoQuota        = errors.New("no available quota")
	ErrSlotAlreadyExists  = errors.New("schedule slot already exists")
	ErrInvalidDate        = errors.New("invalid date format, use YYYY-MM-DD")
	ErrDateInPast         = errors.New("cannot generate slots for past dates")
	ErrSuspensionExists   = errors.New("suspension already exists for this date")
)

type ScheduleService struct {
	slotRepo     repository.ScheduleSlotRepository
	suspensionRepo repository.SuspensionRepository
	doctorRepo   repository.DoctorRepository
	apptRepo     repository.AppointmentRepository
	cfg          *config.AppointmentConfig
}

func NewScheduleService(
	slotRepo repository.ScheduleSlotRepository,
	suspensionRepo repository.SuspensionRepository,
	doctorRepo repository.DoctorRepository,
	apptRepo repository.AppointmentRepository,
	cfg *config.AppointmentConfig,
) *ScheduleService {
	return &ScheduleService{
		slotRepo:       slotRepo,
		suspensionRepo: suspensionRepo,
		doctorRepo:     doctorRepo,
		apptRepo:       apptRepo,
		cfg:            cfg,
	}
}

func (s *ScheduleService) ListSlots(ctx context.Context, doctorID int64, startDate, endDate string) ([]models.ScheduleSlot, error) {
	if startDate == "" {
		startDate = time.Now().Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	}
	return s.slotRepo.ListByDoctorAndDateRange(ctx, doctorID, startDate, endDate)
}

func (s *ScheduleService) GetSlot(ctx context.Context, id int64) (*models.ScheduleSlot, error) {
	slot, err := s.slotRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if slot == nil {
		return nil, ErrSlotNotFound
	}
	return slot, nil
}

func (s *ScheduleService) GenerateSlotsForDate(ctx context.Context, doctorID int64, dateStr string) error {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return ErrInvalidDate
	}

	if date.Before(time.Now().Truncate(24 * time.Hour)) {
		return ErrDateInPast
	}

	doctor, err := s.doctorRepo.GetByID(ctx, doctorID)
	if err != nil {
		return err
	}
	if doctor == nil {
		return ErrDoctorNotFound
	}

	templates, err := s.getTemplatesForDay(ctx, doctorID, date.Weekday())
	if err != nil {
		return err
	}

	for _, tpl := range templates {
		existing, err := s.slotRepo.GetByDoctorDateTime(ctx, doctorID, dateStr, tpl.StartTime, tpl.EndTime)
		if err != nil {
			logger.Error("check existing slot failed", zap.Error(err))
			continue
		}
		if existing != nil {
			continue
		}

		quota := tpl.Quota
		if quota <= 0 {
			quota = s.cfg.DefaultQuotaPerSlot
		}

		slot := &models.ScheduleSlot{
			DoctorID:     doctorID,
			ScheduleDate: dateStr,
			StartTime:    tpl.StartTime,
			EndTime:      tpl.EndTime,
			TotalQuota:   quota,
			UsedQuota:    0,
		}
		if err := s.slotRepo.Create(ctx, slot); err != nil {
			logger.Error("create slot failed", zap.Error(err))
		}
	}

	return nil
}

type templateRow struct {
	ID        int64
	DoctorID  int64
	DayOfWeek string
	StartTime string
	EndTime   string
	Quota     int
	IsActive  bool
}

func (s *ScheduleService) getTemplatesForDay(ctx context.Context, doctorID int64, weekday time.Weekday) ([]templateRow, error) {
	dayMap := map[time.Weekday]string{
		time.Monday:    "monday",
		time.Tuesday:   "tuesday",
		time.Wednesday: "wednesday",
		time.Thursday:  "thursday",
		time.Friday:    "friday",
		time.Saturday:  "saturday",
		time.Sunday:    "sunday",
	}
	dayStr := dayMap[weekday]

	rows, err := db.GetDB().QueryContext(ctx, `
		SELECT id, doctor_id, day_of_week::text, start_time::text, end_time::text, quota, is_active
		FROM schedule_templates
		WHERE doctor_id = $1 AND day_of_week = $2::day_of_week AND is_active = TRUE
	`, doctorID, dayStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []templateRow
	for rows.Next() {
		var t templateRow
		if err := rows.Scan(&t.ID, &t.DoctorID, &t.DayOfWeek, &t.StartTime, &t.EndTime, &t.Quota, &t.IsActive); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

func (s *ScheduleService) CreateSuspension(ctx context.Context, doctorID int64, dateStr, reason string) (int, error) {
	_, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 0, ErrInvalidDate
	}

	exists, err := s.suspensionRepo.Exists(ctx, doctorID, dateStr)
	if err != nil {
		return 0, err
	}
	if exists {
		return 0, ErrSuspensionExists
	}

	tx, err := db.BeginTx(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	suspension := &models.SuspensionDay{
		DoctorID:    doctorID,
		SuspendDate: dateStr,
		Reason:      reason,
	}
	if err := s.suspensionRepo.Create(ctx, tx, suspension); err != nil {
		_ = tx.Rollback()
		return 0, err
	}

	slots, err := s.slotRepo.ListByDoctorAndDateRange(ctx, doctorID, dateStr, dateStr)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}

	totalCancelled := 0
	for _, slot := range slots {
		slot.IsSuspended = true
		_, err := db.GetDB().ExecContext(ctx, `
			UPDATE schedule_slots SET is_suspended = TRUE WHERE id = $1
		`, slot.ID)
		if err != nil {
			_ = tx.Rollback()
			return 0, err
		}

		cancelled, err := s.apptRepo.CancelBySlotID(ctx, tx, slot.ID, "doctor", fmt.Sprintf("doctor suspension: %s", reason))
		if err != nil {
			_ = tx.Rollback()
			return 0, err
		}
		if cancelled > 0 {
			if err := s.slotRepo.UpdateUsedQuota(ctx, tx, slot.ID, -int(cancelled)); err != nil {
				_ = tx.Rollback()
				return 0, err
			}
		}
		totalCancelled += int(cancelled)
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	logger.Info("suspension created",
		zap.Int64("doctor_id", doctorID),
		zap.String("date", dateStr),
		zap.Int("cancelled_appointments", totalCancelled),
	)

	return totalCancelled, nil
}
