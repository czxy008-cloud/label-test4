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
