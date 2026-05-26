package service

import (
	"testing"
	"time"

	"clinic-appointment/internal/config"
	"clinic-appointment/internal/models"
)

func TestGenerateAppointmentNo(t *testing.T) {
	no1 := generateAppointmentNo()
	no2 := generateAppointmentNo()

	if no1 == no2 {
		t.Error("appointment numbers should be unique")
	}
	if len(no1) < 10 {
		t.Error("appointment number too short")
	}
	if len(no2) < 10 {
		t.Error("appointment number too short")
	}
}

func TestValidateCancellationWindow(t *testing.T) {
	cfg := &config.AppointmentConfig{
		CancelBeforeHours: 24,
	}

	svc := &AppointmentService{cfg: cfg}

	tests := []struct {
		name      string
		date      string
		startTime string
		wantErr   bool
	}{
		{
			name:      "future appointment can be cancelled",
			date:      time.Now().Add(48 * time.Hour).Format("2006-01-02"),
			startTime: "09:00",
			wantErr:   false,
		},
		{
			name:      "appointment within 24h cannot be cancelled",
			date:      time.Now().Add(1 * time.Hour).Format("2006-01-02"),
			startTime: "09:00",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slot := &models.ScheduleSlot{
				ScheduleDate: tt.date,
				StartTime:    tt.startTime,
			}
			err := svc.validateCancellationWindow(slot)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCancellationWindow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetDayOfWeekStr(t *testing.T) {
	tests := []struct {
		weekday time.Weekday
		want    string
	}{
		{time.Monday, "monday"},
		{time.Tuesday, "tuesday"},
		{time.Wednesday, "wednesday"},
		{time.Thursday, "thursday"},
		{time.Friday, "friday"},
		{time.Saturday, "saturday"},
		{time.Sunday, "sunday"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			dayMap := map[time.Weekday]string{
				time.Monday:    "monday",
				time.Tuesday:   "tuesday",
				time.Wednesday: "wednesday",
				time.Thursday:  "thursday",
				time.Friday:    "friday",
				time.Saturday:  "saturday",
				time.Sunday:    "sunday",
			}
			if got := dayMap[tt.weekday]; got != tt.want {
				t.Errorf("getDayOfWeekStr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWaitlistStatusConstants(t *testing.T) {
	tests := []struct {
		status models.WaitlistStatus
		want   string
	}{
		{models.WaitlistStatusWaiting, "waiting"},
		{models.WaitlistStatusPromoted, "promoted"},
		{models.WaitlistStatusCancelled, "cancelled"},
		{models.WaitlistStatusExpired, "expired"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("WaitlistStatus constant = %v, want %v", tt.status, tt.want)
			}
		})
	}
}

func TestNotificationTypeConstants(t *testing.T) {
	tests := []struct {
		notifType models.NotificationType
		want      string
	}{
		{models.NotificationWaitlistPromoted, "waitlist_promoted"},
		{models.NotificationApptCancelled, "appointment_cancelled"},
		{models.NotificationApptConfirmed, "appointment_confirmed"},
		{models.NotificationApptSuspended, "appointment_suspended"},
	}

	for _, tt := range tests {
		t.Run(string(tt.notifType), func(t *testing.T) {
			if string(tt.notifType) != tt.want {
				t.Errorf("NotificationType constant = %v, want %v", tt.notifType, tt.want)
			}
		})
	}
}

func TestWaitlistModel(t *testing.T) {
	now := time.Now()
	apptID := int64(123)

	wl := &models.Waitlist{
		ID:            1,
		SlotID:        100,
		PatientName:   "张三",
		PatientPhone:  "13800138000",
		PatientIDCard: "110101199001011234",
		Status:        models.WaitlistStatusWaiting,
		Position:      1,
		AppointmentID: &apptID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if wl.ID != 1 {
		t.Errorf("Waitlist ID = %v, want 1", wl.ID)
	}
	if wl.PatientName != "张三" {
		t.Errorf("Waitlist PatientName = %v, want 张三", wl.PatientName)
	}
	if wl.Position != 1 {
		t.Errorf("Waitlist Position = %v, want 1", wl.Position)
	}
	if *wl.AppointmentID != 123 {
		t.Errorf("Waitlist AppointmentID = %v, want 123", *wl.AppointmentID)
	}
	if wl.Status != models.WaitlistStatusWaiting {
		t.Errorf("Waitlist Status = %v, want waiting", wl.Status)
	}
}

func TestNotificationModel(t *testing.T) {
	now := time.Now()

	notif := &models.Notification{
		ID:             1,
		Type:           models.NotificationWaitlistPromoted,
		RecipientPhone: "13800138000",
		RecipientName:  "张三",
		Content:        "您的候补已递补成功",
		Metadata: map[string]string{
			"slot_id": "100",
		},
		IsProcessed: false,
		CreatedAt:   now,
	}

	if notif.ID != 1 {
		t.Errorf("Notification ID = %v, want 1", notif.ID)
	}
	if notif.Type != models.NotificationWaitlistPromoted {
		t.Errorf("Notification Type = %v, want waitlist_promoted", notif.Type)
	}
	if notif.Content != "您的候补已递补成功" {
		t.Errorf("Notification Content = %v, want 您的候补已递补成功", notif.Content)
	}
	if notif.Metadata["slot_id"] != "100" {
		t.Errorf("Notification Metadata slot_id = %v, want 100", notif.Metadata["slot_id"])
	}
	if notif.IsProcessed != false {
		t.Errorf("Notification IsProcessed = %v, want false", notif.IsProcessed)
	}
}

func TestGenerateAppointmentNoFormat(t *testing.T) {
	no := generateAppointmentNo()
	todayPrefix := "APT" + time.Now().Format("20060102")

	if len(no) != len(todayPrefix)+8 {
		t.Errorf("appointment no length = %d, want %d", len(no), len(todayPrefix)+8)
	}
	if no[:len(todayPrefix)] != todayPrefix {
		t.Errorf("appointment no prefix = %s, want %s", no[:len(todayPrefix)], todayPrefix)
	}
}
