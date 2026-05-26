package service

import (
	"context"
	"database/sql"
	"fmt"

	"clinic-appointment/internal/db"
	"clinic-appointment/internal/logger"
	"clinic-appointment/internal/models"
	"clinic-appointment/internal/repository"

	"go.uber.org/zap"
)

type NotificationService struct {
	notifRepo repository.NotificationRepository
}

func NewNotificationService(notifRepo repository.NotificationRepository) *NotificationService {
	return &NotificationService{
		notifRepo: notifRepo,
	}
}

func (s *NotificationService) NotifyWaitlistPromoted(ctx context.Context, tx *sql.Tx, wl *models.Waitlist, apptNo string) error {
	slot := wl.Slot
	doctorName := ""
	if slot != nil && slot.Doctor != nil {
		doctorName = slot.Doctor.Name
	}

	content := fmt.Sprintf("您好，%s！您预约的%s %s %s号源已有空位，已为您自动递补成功，预约单号：%s，请及时确认。",
		wl.PatientName, slot.ScheduleDate, slot.StartTime, doctorName, apptNo)

	notification := &models.Notification{
		Type:           models.NotificationWaitlistPromoted,
		RecipientPhone: wl.PatientPhone,
		RecipientName:  wl.PatientName,
		Content:        content,
		Metadata: map[string]string{
			"slot_id":        fmt.Sprintf("%d", wl.SlotID),
			"waitlist_id":    fmt.Sprintf("%d", wl.ID),
			"appointment_no": apptNo,
			"position":       fmt.Sprintf("%d", wl.Position),
		},
	}

	if err := s.notifRepo.Enqueue(ctx, tx, notification); err != nil {
		logger.Error("failed to enqueue waitlist promoted notification",
			zap.Error(err),
			zap.Int64("waitlist_id", wl.ID),
			zap.String("patient_phone", wl.PatientPhone),
		)
		return err
	}

	logger.Info("waitlist promoted notification enqueued",
		zap.Int64("notification_id", notification.ID),
		zap.Int64("waitlist_id", wl.ID),
		zap.String("patient_phone", wl.PatientPhone),
	)

	return nil
}

func (s *NotificationService) NotifyAppointmentCancelled(ctx context.Context, tx *sql.Tx, appt *models.Appointment, reason string) error {
	slot := appt.Slot
	doctorName := ""
	if slot != nil && slot.Doctor != nil {
		doctorName = slot.Doctor.Name
	}

	content := fmt.Sprintf("您好，%s！您预约的%s %s %s的预约已取消，原因：%s。",
		appt.PatientName, slot.ScheduleDate, slot.StartTime, doctorName, reason)

	notification := &models.Notification{
		Type:           models.NotificationApptCancelled,
		RecipientPhone: appt.PatientPhone,
		RecipientName:  appt.PatientName,
		Content:        content,
		Metadata: map[string]string{
			"slot_id":         fmt.Sprintf("%d", appt.SlotID),
			"appointment_id":  fmt.Sprintf("%d", appt.ID),
			"appointment_no":  appt.AppointmentNo,
			"cancel_reason":   reason,
		},
	}

	if err := s.notifRepo.Enqueue(ctx, tx, notification); err != nil {
		logger.Error("failed to enqueue appointment cancelled notification",
			zap.Error(err),
			zap.Int64("appointment_id", appt.ID),
			zap.String("patient_phone", appt.PatientPhone),
		)
		return err
	}

	logger.Info("appointment cancelled notification enqueued",
		zap.Int64("notification_id", notification.ID),
		zap.Int64("appointment_id", appt.ID),
		zap.String("patient_phone", appt.PatientPhone),
	)

	return nil
}

func (s *NotificationService) NotifyAppointmentSuspended(ctx context.Context, tx *sql.Tx, appt *models.Appointment, reason string) error {
	slot := appt.Slot
	doctorName := ""
	if slot != nil && slot.Doctor != nil {
		doctorName = slot.Doctor.Name
	}

	content := fmt.Sprintf("您好，%s！医生临时停诊，您预约的%s %s %s的预约已取消，原因：%s。如有需要可重新预约其他时间。",
		appt.PatientName, slot.ScheduleDate, slot.StartTime, doctorName, reason)

	notification := &models.Notification{
		Type:           models.NotificationApptSuspended,
		RecipientPhone: appt.PatientPhone,
		RecipientName:  appt.PatientName,
		Content:        content,
		Metadata: map[string]string{
			"slot_id":         fmt.Sprintf("%d", appt.SlotID),
			"appointment_id":  fmt.Sprintf("%d", appt.ID),
			"appointment_no":  appt.AppointmentNo,
			"suspend_reason":  reason,
		},
	}

	if err := s.notifRepo.Enqueue(ctx, tx, notification); err != nil {
		logger.Error("failed to enqueue appointment suspended notification",
			zap.Error(err),
			zap.Int64("appointment_id", appt.ID),
			zap.String("patient_phone", appt.PatientPhone),
		)
		return err
	}

	logger.Info("appointment suspended notification enqueued",
		zap.Int64("notification_id", notification.ID),
		zap.Int64("appointment_id", appt.ID),
		zap.String("patient_phone", appt.PatientPhone),
	)

	return nil
}

func (s *NotificationService) ProcessPending(ctx context.Context, limit int) (int, error) {
	notifications, err := s.notifRepo.GetPending(ctx, limit)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, notif := range notifications {
		if err := s.sendNotification(ctx, &notif); err != nil {
			logger.Error("failed to send notification",
				zap.Error(err),
				zap.Int64("notification_id", notif.ID),
				zap.String("type", string(notif.Type)),
			)
			continue
		}

		if err := s.notifRepo.MarkProcessed(ctx, notif.ID); err != nil {
			logger.Error("failed to mark notification as processed",
				zap.Error(err),
				zap.Int64("notification_id", notif.ID),
			)
			continue
		}

		processed++
		logger.Info("notification processed successfully",
			zap.Int64("notification_id", notif.ID),
			zap.String("type", string(notif.Type)),
		)
	}

	return processed, nil
}

func (s *NotificationService) sendNotification(ctx context.Context, notif *models.Notification) error {
	logger.Info("sending notification",
		zap.String("type", string(notif.Type)),
		zap.String("recipient", notif.RecipientPhone),
		zap.String("content", notif.Content),
	)
	return nil
}

func (s *NotificationService) StartWorker(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Info("notification worker stopped")
				return
			default:
				if _, err := s.ProcessPending(ctx, 10); err != nil {
					logger.Error("notification worker error", zap.Error(err))
				}
				// Wait for a short period before checking again
				// In production, this would use proper message queue polling
				select {
				case <-ctx.Done():
					return
				}
			}
		}
	}()
}

func ProcessNotificationsManually() {
	ctx := context.Background()
	notifRepo := repository.NewNotificationRepository()
	notifService := NewNotificationService(notifRepo)

	processed, err := notifService.ProcessPending(ctx, 100)
	if err != nil {
		logger.Error("manual notification processing failed", zap.Error(err))
		return
	}
	logger.Info("manual notification processing completed", zap.Int("processed", processed))
}

func BeginTx(ctx context.Context) (*sql.Tx, error) {
	return db.BeginTx(ctx)
}
