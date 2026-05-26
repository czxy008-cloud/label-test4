package service

import (
	"testing"

	"clinic-appointment/internal/models"
)

func TestConflictTypeConstants(t *testing.T) {
	if string(models.ConflictTypeDuplicateSlot) != "duplicate_slot" {
		t.Errorf("ConflictTypeDuplicateSlot = %v, want duplicate_slot", models.ConflictTypeDuplicateSlot)
	}
	if string(models.ConflictTypeExistingAppt) != "existing_appointment" {
		t.Errorf("ConflictTypeExistingAppt = %v, want existing_appointment", models.ConflictTypeExistingAppt)
	}
}

func TestScheduleConflictResponseEmpty(t *testing.T) {
	resp := &models.ScheduleConflictResponse{
		HasConflict: false,
		Conflicts:   []models.ScheduleConflict{},
	}
	if resp.HasConflict {
		t.Error("expected HasConflict to be false")
	}
	if len(resp.Conflicts) != 0 {
		t.Errorf("expected 0 conflicts, got %d", len(resp.Conflicts))
	}
}

func TestScheduleConflictResponseWithDuplicateSlot(t *testing.T) {
	existing := &models.ScheduleSlot{
		ID:           1,
		DoctorID:     10,
		ScheduleDate: "2026-06-01",
		StartTime:    "09:00",
		EndTime:      "10:00",
		TotalQuota:   10,
	}
	conflict := models.ScheduleConflict{
		Type:         models.ConflictTypeDuplicateSlot,
		Date:         "2026-06-01",
		StartTime:    "09:00",
		EndTime:      "10:00",
		Message:      "09:00-10:00 已存在排班号源",
		ExistingSlot: existing,
	}
	if conflict.Type != models.ConflictTypeDuplicateSlot {
		t.Errorf("conflict type = %v, want duplicate_slot", conflict.Type)
	}
	if conflict.ExistingSlot == nil || conflict.ExistingSlot.ID != 1 {
		t.Error("ExistingSlot should be populated with the conflicting slot")
	}
	if conflict.Message != "09:00-10:00 已存在排班号源" {
		t.Errorf("conflict message = %v", conflict.Message)
	}
}

func TestScheduleConflictResponseWithExistingAppointment(t *testing.T) {
	conflict := models.ScheduleConflict{
		Type:      models.ConflictTypeExistingAppt,
		Date:      "2026-06-01",
		StartTime: "10:00",
		EndTime:   "10:30",
		Message:   "10:00-10:30 已有 3 个有效预约",
		ApptCount: 3,
	}
	if conflict.ApptCount != 3 {
		t.Errorf("ApptCount = %v, want 3", conflict.ApptCount)
	}
	if conflict.Message != "10:00-10:30 已有 3 个有效预约" {
		t.Errorf("Message = %v", conflict.Message)
	}
}

func TestAggregateAppointmentsByRange(t *testing.T) {
	type rangeKey struct {
		start, end string
	}
	appts := []models.Appointment{
		{ID: 1, Status: models.StatusPending, Slot: &models.ScheduleSlot{StartTime: "10:00", EndTime: "10:30"}},
		{ID: 2, Status: models.StatusConfirmed, Slot: &models.ScheduleSlot{StartTime: "10:00", EndTime: "10:30"}},
		{ID: 3, Status: models.StatusPending, Slot: &models.ScheduleSlot{StartTime: "10:00", EndTime: "10:30"}},
		{ID: 4, Status: models.StatusConfirmed, Slot: &models.ScheduleSlot{StartTime: "11:00", EndTime: "11:30"}},
	}
	counts := map[rangeKey]int{}
	for _, appt := range appts {
		if appt.Slot == nil {
			continue
		}
		key := rangeKey{start: appt.Slot.StartTime, end: appt.Slot.EndTime}
		counts[key]++
	}
	if counts[rangeKey{"10:00", "10:30"}] != 3 {
		t.Errorf("expected 3 appointments in 10:00-10:30, got %d", counts[rangeKey{"10:00", "10:30"}])
	}
	if counts[rangeKey{"11:00", "11:30"}] != 1 {
		t.Errorf("expected 1 appointment in 11:00-11:30, got %d", counts[rangeKey{"11:00", "11:30"}])
	}
}
