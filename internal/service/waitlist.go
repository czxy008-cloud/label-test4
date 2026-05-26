package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"clinic-appointment/internal/config"
	"clinic-appointment/internal/db"
	"clinic-appointment/internal/dto"
	"clinic-appointment/internal/logger"
	"clinic-appointment/internal/models"
	"clinic-appointment/internal/repository"

	"go.uber.org/zap"
)

var (
	ErrWaitlistNotFound  = errors.New("waitlist entry not found")
	ErrWaitlistNotActive = errors.New("waitlist entry is not active")
	ErrAlreadyInWaitlist = errors.New("patient is already in the waitlist for this slot")
	ErrSlotNotFull       = errors.New("slot is not full, can join directly")
	ErrSlotSuspendedWaitlist = errors.New("slot is suspended, cannot join waitlist")
)

type WaitlistService struct {
	waitlistRepo  repository.WaitlistRepository
	apptRepo      repository.AppointmentRepository
	logRepo       repository.AppointmentLogRepository
	slotRepo      repository.ScheduleSlotRepository
	notifService  *NotificationService
	cfg           *config.AppointmentConfig
}

func NewWaitlistService(
	waitlistRepo repository.WaitlistRepository,
	apptRepo repository.AppointmentRepository,
	logRepo repository.AppointmentLogRepository,
	slotRepo repository.ScheduleSlotRepository,
	notifService *NotificationService,
	cfg *config.AppointmentConfig,
) *WaitlistService {
	return &WaitlistService{
		waitlistRepo: waitlistRepo,
		apptRepo:     apptRepo,
		logRepo:      logRepo,
		slotRepo:     slotRepo,
		notifService: notifService,
		cfg:          cfg,
	}
}

func (s *WaitlistService) JoinWaitlist(ctx context.Context, slotID int64, patientName, patientPhone, patientIDCard string) (*models.Waitlist, error) {
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
		return nil, ErrSlotSuspendedWaitlist
	}

	if slot.UsedQuota < slot.TotalQuota {
		_ = tx.Rollback()
		return nil, ErrSlotNotFull
	}

	exists, err := s.waitlistRepo.ExistsActiveBySlotAndPhone(ctx, slotID, patientPhone)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if exists {
		_ = tx.Rollback()
		return nil, ErrAlreadyInWaitlist
	}

	var existingAppt int64
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM appointments
		WHERE slot_id = $1 AND patient_phone = $2 AND status IN ('pending', 'confirmed')
	`, slotID, patientPhone).Scan(&existingAppt)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if existingAppt > 0 {
		_ = tx.Rollback()
		return nil, ErrDuplicateAppointment
	}

	position, err := s.waitlistRepo.GetNextPosition(ctx, slotID)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	wl := &models.Waitlist{
		SlotID:        slotID,
		PatientName:   patientName,
		PatientPhone:  patientPhone,
		PatientIDCard: patientIDCard,
		Status:        models.WaitlistStatusWaiting,
		Position:      position,
	}

	if err := s.waitlistRepo.Create(ctx, tx, wl); err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	wl.Slot = &slot

	logger.Info("patient joined waitlist",
		zap.Int64("waitlist_id", wl.ID),
		zap.Int64("slot_id", slotID),
		zap.String("patient_phone", patientPhone),
		zap.Int("position", position),
	)

	return wl, nil
}

func (s *WaitlistService) GetWaitlist(ctx context.Context, id int64) (*models.Waitlist, error) {
	wl, err := s.waitlistRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if wl == nil {
		return nil, ErrWaitlistNotFound
	}
	return wl, nil
}

func (s *WaitlistService) CancelWaitlist(ctx context.Context, id int64, reason string) error {
	wl, err := s.waitlistRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if wl == nil {
		return ErrWaitlistNotFound
	}

	if wl.Status != models.WaitlistStatusWaiting {
		return ErrWaitlistNotActive
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

	if err := s.waitlistRepo.UpdateStatus(ctx, tx, id, models.WaitlistStatusCancelled, nil); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	logger.Info("waitlist cancelled",
		zap.Int64("waitlist_id", id),
		zap.String("reason", reason),
	)

	return nil
}

func (s *WaitlistService) ListByPatient(ctx context.Context, patientPhone string, status models.WaitlistStatus, page, pageSize int) ([]models.Waitlist, int64, error) {
	offset := (page - 1) * pageSize
	return s.waitlistRepo.ListByPatient(ctx, patientPhone, status, offset, pageSize)
}

func (s *WaitlistService) PromoteFromWaitlist(ctx context.Context, tx *sql.Tx, slotID int64) (*dto.PromoteResult, error) {
	nextWaitlist, err := s.waitlistRepo.GetNextWaiting(ctx, tx, slotID)
	if err != nil {
		return nil, err
	}
	if nextWaitlist == nil {
		return nil, nil
	}

	slot, err := s.slotRepo.GetByID(ctx, slotID)
	if err != nil {
		return nil, err
	}
	if slot == nil {
		return nil, ErrSlotNotFound
	}

	apptNo := generateAppointmentNo()

	appt := &models.Appointment{
		SlotID:        slotID,
		PatientName:   nextWaitlist.PatientName,
		PatientPhone:  nextWaitlist.PatientPhone,
		PatientIDCard: nextWaitlist.PatientIDCard,
		Status:        models.StatusPending,
		AppointmentNo: apptNo,
	}

	if err := s.apptRepo.Create(ctx, tx, appt); err != nil {
		return nil, err
	}

	if err := s.slotRepo.UpdateUsedQuota(ctx, tx, slotID, 1); err != nil {
		return nil, err
	}

	if err := s.waitlistRepo.UpdateStatus(ctx, tx, nextWaitlist.ID, models.WaitlistStatusPromoted, &appt.ID); err != nil {
		return nil, err
	}

	logEntry := &models.AppointmentLog{
		AppointmentID: appt.ID,
		NewStatus:     models.StatusPending,
		Operator:      "system",
		Reason:        fmt.Sprintf("promoted from waitlist, position: %d", nextWaitlist.Position),
	}
	if err := s.logRepo.Create(ctx, logEntry); err != nil {
		return nil, err
	}

	nextWaitlist.Slot = slot
	if err := s.notifService.NotifyWaitlistPromoted(ctx, tx, nextWaitlist, apptNo); err != nil {
		logger.Warn("failed to enqueue promotion notification, but promotion succeeded",
			zap.Error(err),
			zap.Int64("waitlist_id", nextWaitlist.ID),
		)
	}

	logger.Info("waitlist patient promoted successfully",
		zap.Int64("waitlist_id", nextWaitlist.ID),
		zap.Int64("appointment_id", appt.ID),
		zap.String("appointment_no", apptNo),
		zap.Int("position", nextWaitlist.Position),
		zap.Int64("slot_id", slotID),
	)

	return &dto.PromoteResult{
		WaitlistID:    nextWaitlist.ID,
		AppointmentID: appt.ID,
		AppointmentNo: apptNo,
		PatientName:   nextWaitlist.PatientName,
		PatientPhone:  nextWaitlist.PatientPhone,
		Position:      nextWaitlist.Position,
	}, nil
}

func (s *WaitlistService) GetWaitlistCount(ctx context.Context, slotID int64) (int64, error) {
	return s.waitlistRepo.CountWaitingBySlotID(ctx, slotID)
}
