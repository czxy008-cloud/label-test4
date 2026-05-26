package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"clinic-appointment/internal/config"
	"clinic-appointment/internal/db"
	"clinic-appointment/internal/dto"
	"clinic-appointment/internal/logger"
	"clinic-appointment/internal/models"
	"clinic-appointment/internal/repository"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrAppointmentNotFound    = errors.New("appointment not found")
	ErrInvalidStatus          = errors.New("invalid appointment status")
	ErrCannotCancelAfterStart = errors.New("cannot cancel appointment after start time")
	ErrCannotCancelPastWindow = errors.New("cannot cancel appointment, cancellation window has passed")
	ErrAlreadyCancelled       = errors.New("appointment already cancelled")
	ErrDuplicateAppointment   = errors.New("patient already has an appointment for this slot")
)

type AppointmentService struct {
	apptRepo      repository.AppointmentRepository
	logRepo       repository.AppointmentLogRepository
	slotRepo      repository.ScheduleSlotRepository
	waitlistRepo  repository.WaitlistRepository
	waitlistService *WaitlistService
	notifService  *NotificationService
	cfg           *config.AppointmentConfig
}

func NewAppointmentService(
	apptRepo repository.AppointmentRepository,
	logRepo repository.AppointmentLogRepository,
	slotRepo repository.ScheduleSlotRepository,
	waitlistRepo repository.WaitlistRepository,
	notifService *NotificationService,
	cfg *config.AppointmentConfig,
) *AppointmentService {
	return &AppointmentService{
		apptRepo:     apptRepo,
		logRepo:      logRepo,
		slotRepo:     slotRepo,
		waitlistRepo: waitlistRepo,
		notifService: notifService,
		cfg:          cfg,
	}
}

func (s *AppointmentService) SetWaitlistService(wlService *WaitlistService) {
	s.waitlistService = wlService
}

func (s *AppointmentService) CreateAppointment(ctx context.Context, slotID int64, patientName, patientPhone, patientIDCard string) (*models.Appointment, error) {
	tx, err := db.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	var slot models.ScheduleSlot
	err = tx.QueryRowContext(ctx, `
		SELECT id, doctor_id, schedule_date, start_time, end_time, total_quota, used_quota, is_suspended
		FROM schedule_slots
		WHERE id = $1
		FOR UPDATE
	`, slotID).Scan(
		&slot.ID, &slot.DoctorID, &slot.ScheduleDate, &slot.StartTime, &slot.EndTime,
		&slot.TotalQuota, &slot.UsedQuota, &slot.IsSuspended,
	)
	if err == sql.ErrNoRows {
		_ = tx.Rollback()
		return nil, ErrSlotNotFound
	}
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if slot.IsSuspended {
		_ = tx.Rollback()
		return nil, ErrSlotSuspended
	}

	if slot.UsedQuota >= slot.TotalQuota {
		_ = tx.Rollback()
		return nil, ErrSlotNoQuota
	}

	var existing int64
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM appointments
		WHERE slot_id = $1 AND patient_phone = $2 AND status IN ('pending', 'confirmed')
	`, slotID, patientPhone).Scan(&existing)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if existing > 0 {
		_ = tx.Rollback()
		return nil, ErrDuplicateAppointment
	}

	apptNo := generateAppointmentNo()

	appt := &models.Appointment{
		SlotID:        slotID,
		PatientName:   patientName,
		PatientPhone:  patientPhone,
		PatientIDCard: patientIDCard,
		Status:        models.StatusPending,
		AppointmentNo: apptNo,
	}

	if err := s.apptRepo.Create(ctx, tx, appt); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := s.slotRepo.UpdateUsedQuota(ctx, tx, slotID, 1); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	logEntry := &models.AppointmentLog{
		AppointmentID: appt.ID,
		NewStatus:     models.StatusPending,
		Operator:      "patient",
		Reason:        "create appointment",
	}
	if err := s.logRepo.Create(ctx, logEntry); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	logger.Info("appointment created",
		zap.Int64("appointment_id", appt.ID),
		zap.String("appointment_no", apptNo),
		zap.Int64("slot_id", slotID),
		zap.String("patient_phone", patientPhone),
	)

	return appt, nil
}

func (s *AppointmentService) GetAppointment(ctx context.Context, id int64) (*models.Appointment, error) {
	appt, err := s.apptRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if appt == nil {
		return nil, ErrAppointmentNotFound
	}
	return appt, nil
}

func (s *AppointmentService) GetAppointmentByNo(ctx context.Context, no string) (*models.Appointment, error) {
	appt, err := s.apptRepo.GetByNo(ctx, no)
	if err != nil {
		return nil, err
	}
	if appt == nil {
		return nil, ErrAppointmentNotFound
	}
	return appt, nil
}

func (s *AppointmentService) CancelAppointment(ctx context.Context, id int64, reason string) (*dto.PromoteResult, error) {
	appt, err := s.apptRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if appt == nil {
		return nil, ErrAppointmentNotFound
	}

	if appt.Status == models.StatusCancelled || appt.Status == models.StatusSuspended {
		return nil, ErrAlreadyCancelled
	}

	slot, err := s.slotRepo.GetByID(ctx, appt.SlotID)
	if err != nil {
		return nil, err
	}
	if slot == nil {
		return nil, ErrSlotNotFound
	}

	if err := s.validateCancellationWindow(slot); err != nil {
		return nil, err
	}

	tx, err := db.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := s.apptRepo.UpdateStatus(ctx, tx, id, models.StatusCancelled, "patient", reason); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := s.slotRepo.UpdateUsedQuota(ctx, tx, appt.SlotID, -1); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := s.notifService.NotifyAppointmentCancelled(ctx, tx, appt, reason); err != nil {
		logger.Warn("failed to enqueue cancellation notification, but cancellation succeeded",
			zap.Error(err),
			zap.Int64("appointment_id", id),
		)
	}

	var promoteResult *dto.PromoteResult
	if s.waitlistService != nil {
		promoteResult, err = s.waitlistService.PromoteFromWaitlist(ctx, tx, appt.SlotID)
		if err != nil {
			_ = tx.Rollback()
			return nil, fmt.Errorf("failed to promote from waitlist: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	logger.Info("appointment cancelled",
		zap.Int64("appointment_id", id),
		zap.String("reason", reason),
		zap.Bool("waitlist_promoted", promoteResult != nil),
	)

	return promoteResult, nil
}

func (s *AppointmentService) validateCancellationWindow(slot *models.ScheduleSlot) error {
	apptDateTime, err := time.ParseInLocation("2006-01-02 15:04", fmt.Sprintf("%s %s", slot.ScheduleDate, slot.StartTime), time.Local)
	if err != nil {
		return err
	}

	cancelDeadline := apptDateTime.Add(-time.Duration(s.cfg.CancelBeforeHours) * time.Hour)
	if time.Now().After(cancelDeadline) {
		return ErrCannotCancelPastWindow
	}

	return nil
}

func (s *AppointmentService) ConfirmAppointment(ctx context.Context, id int64) error {
	appt, err := s.apptRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if appt == nil {
		return ErrAppointmentNotFound
	}

	if appt.Status != models.StatusPending {
		return ErrInvalidStatus
	}

	tx, err := db.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := s.apptRepo.UpdateStatus(ctx, tx, id, models.StatusConfirmed, "clinic", "patient checked in"); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (s *AppointmentService) CompleteAppointment(ctx context.Context, id int64) error {
	appt, err := s.apptRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if appt == nil {
		return ErrAppointmentNotFound
	}

	if appt.Status != models.StatusConfirmed {
		return ErrInvalidStatus
	}

	tx, err := db.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := s.apptRepo.UpdateStatus(ctx, tx, id, models.StatusCompleted, "doctor", "visit completed"); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (s *AppointmentService) ListByPatient(ctx context.Context, patientPhone string, status models.AppointmentStatus, date string, page, pageSize int) ([]models.Appointment, int64, error) {
	offset := (page - 1) * pageSize
	return s.apptRepo.ListByPatient(ctx, patientPhone, status, date, offset, pageSize)
}

func (s *AppointmentService) ListByDoctor(ctx context.Context, doctorID int64, date string, status models.AppointmentStatus, page, pageSize int) ([]models.Appointment, int64, error) {
	offset := (page - 1) * pageSize
	return s.apptRepo.ListByDoctorAndDate(ctx, doctorID, date, status, offset, pageSize)
}

func (s *AppointmentService) GetAppointmentLogs(ctx context.Context, appointmentID int64) ([]models.AppointmentLog, error) {
	return s.logRepo.ListByAppointmentID(ctx, appointmentID)
}

func generateAppointmentNo() string {
	return "APT" + time.Now().Format("20060102") + uuid.New().String()[:8]
}
