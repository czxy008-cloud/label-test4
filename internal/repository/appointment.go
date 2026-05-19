package repository

import (
	"context"
	"database/sql"
	"strings"

	"clinic-appointment/internal/db"
	"clinic-appointment/internal/models"
)

type appointmentRepo struct{}

func NewAppointmentRepository() AppointmentRepository {
	return &appointmentRepo{}
}

func (r *appointmentRepo) Create(ctx context.Context, tx *sql.Tx, appt *models.Appointment) error {
	return tx.QueryRowContext(ctx, `
		INSERT INTO appointments (slot_id, patient_name, patient_phone, patient_id_card, status, appointment_no)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`, appt.SlotID, appt.PatientName, appt.PatientPhone, appt.PatientIDCard, appt.Status, appt.AppointmentNo,
	).Scan(&appt.ID, &appt.CreatedAt, &appt.UpdatedAt)
}

func (r *appointmentRepo) GetByID(ctx context.Context, id int64) (*models.Appointment, error) {
	var appt models.Appointment
	var slot models.ScheduleSlot
	var doc models.Doctor
	err := db.GetDB().QueryRowContext(ctx, `
		SELECT a.id, a.slot_id, a.patient_name, a.patient_phone, a.patient_id_card,
		       a.status, a.appointment_no, a.created_at, a.updated_at,
		       s.id, s.schedule_date, s.start_time, s.end_time, s.doctor_id,
		       d.name
		FROM appointments a
		LEFT JOIN schedule_slots s ON s.id = a.slot_id
		LEFT JOIN doctors d ON d.id = s.doctor_id
		WHERE a.id = $1
	`, id).Scan(
		&appt.ID, &appt.SlotID, &appt.PatientName, &appt.PatientPhone, &appt.PatientIDCard,
		&appt.Status, &appt.AppointmentNo, &appt.CreatedAt, &appt.UpdatedAt,
		&slot.ID, &slot.ScheduleDate, &slot.StartTime, &slot.EndTime, &slot.DoctorID,
		&doc.Name,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	slot.Doctor = &doc
	appt.Slot = &slot
	return &appt, nil
}

func (r *appointmentRepo) GetByNo(ctx context.Context, appointmentNo string) (*models.Appointment, error) {
	var appt models.Appointment
	var slot models.ScheduleSlot
	var doc models.Doctor
	err := db.GetDB().QueryRowContext(ctx, `
		SELECT a.id, a.slot_id, a.patient_name, a.patient_phone, a.patient_id_card,
		       a.status, a.appointment_no, a.created_at, a.updated_at,
		       s.id, s.schedule_date, s.start_time, s.end_time, s.doctor_id,
		       d.name
		FROM appointments a
		LEFT JOIN schedule_slots s ON s.id = a.slot_id
		LEFT JOIN doctors d ON d.id = s.doctor_id
		WHERE a.appointment_no = $1
	`, appointmentNo).Scan(
		&appt.ID, &appt.SlotID, &appt.PatientName, &appt.PatientPhone, &appt.PatientIDCard,
		&appt.Status, &appt.AppointmentNo, &appt.CreatedAt, &appt.UpdatedAt,
		&slot.ID, &slot.ScheduleDate, &slot.StartTime, &slot.EndTime, &slot.DoctorID,
		&doc.Name,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	slot.Doctor = &doc
	appt.Slot = &slot
	return &appt, nil
}

func (r *appointmentRepo) UpdateStatus(ctx context.Context, tx *sql.Tx, id int64, status models.AppointmentStatus, operator, reason string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE appointments SET status = $1 WHERE id = $2
	`, status, id)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO appointment_logs (appointment_id, new_status, operator, reason)
		VALUES ($1, $2, $3, $4)
	`, id, status, operator, reason)
	return err
}

func (r *appointmentRepo) ListByPatient(ctx context.Context, patientPhone string, status models.AppointmentStatus, date string, offset, limit int) ([]models.Appointment, int64, error) {
	where := []string{"a.patient_phone = $1"}
	args := []interface{}{patientPhone}
	argIdx := 2

	if status != "" {
		where = append(where, "a.status = $"+string(rune('0'+argIdx)))
		args = append(args, status)
		argIdx++
	}
	if date != "" {
		where = append(where, "s.schedule_date = $"+string(rune('0'+argIdx)))
		args = append(args, date)
		argIdx++
	}

	whereSQL := strings.Join(where, " AND ")

	var total int64
	err := db.GetDB().QueryRowContext(ctx, `
		SELECT COUNT(*) FROM appointments a
		LEFT JOIN schedule_slots s ON s.id = a.slot_id
		WHERE `+whereSQL, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT a.id, a.slot_id, a.patient_name, a.patient_phone, a.status, a.appointment_no, a.created_at,
		       s.schedule_date, s.start_time, s.end_time, d.name
		FROM appointments a
		LEFT JOIN schedule_slots s ON s.id = a.slot_id
		LEFT JOIN doctors d ON d.id = s.doctor_id
		WHERE ` + whereSQL + `
		ORDER BY a.created_at DESC
		LIMIT $` + string(rune('0'+argIdx)) + ` OFFSET $` + string(rune('0'+argIdx+1))
	args = append(args, limit, offset)

	rows, err := db.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var appts []models.Appointment
	for rows.Next() {
		var a models.Appointment
		var s models.ScheduleSlot
		var d models.Doctor
		err := rows.Scan(
			&a.ID, &a.SlotID, &a.PatientName, &a.PatientPhone, &a.Status, &a.AppointmentNo, &a.CreatedAt,
			&s.ScheduleDate, &s.StartTime, &s.EndTime, &d.Name,
		)
		if err != nil {
			return nil, 0, err
		}
		s.Doctor = &d
		a.Slot = &s
		appts = append(appts, a)
	}
	return appts, total, rows.Err()
}

func (r *appointmentRepo) ListBySlotIDAndStatus(ctx context.Context, slotID int64, status models.AppointmentStatus) ([]models.Appointment, error) {
	rows, err := db.GetDB().QueryContext(ctx, `
		SELECT id, slot_id, patient_name, patient_phone, status, appointment_no, created_at
		FROM appointments
		WHERE slot_id = $1 AND status = $2
	`, slotID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var appts []models.Appointment
	for rows.Next() {
		var a models.Appointment
		err := rows.Scan(&a.ID, &a.SlotID, &a.PatientName, &a.PatientPhone, &a.Status, &a.AppointmentNo, &a.CreatedAt)
		if err != nil {
			return nil, err
		}
		appts = append(appts, a)
	}
	return appts, rows.Err()
}

func (r *appointmentRepo) ListByDoctorAndDate(ctx context.Context, doctorID int64, date string, status models.AppointmentStatus, offset, limit int) ([]models.Appointment, int64, error) {
	where := []string{"s.doctor_id = $1"}
	args := []interface{}{doctorID}
	argIdx := 2

	if date != "" {
		where = append(where, "s.schedule_date = $"+string(rune('0'+argIdx)))
		args = append(args, date)
		argIdx++
	}
	if status != "" {
		where = append(where, "a.status = $"+string(rune('0'+argIdx)))
		args = append(args, status)
		argIdx++
	}

	whereSQL := strings.Join(where, " AND ")

	var total int64
	err := db.GetDB().QueryRowContext(ctx, `
		SELECT COUNT(*) FROM appointments a
		LEFT JOIN schedule_slots s ON s.id = a.slot_id
		WHERE `+whereSQL, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT a.id, a.slot_id, a.patient_name, a.patient_phone, a.status, a.appointment_no, a.created_at,
		       s.schedule_date, s.start_time, s.end_time
		FROM appointments a
		LEFT JOIN schedule_slots s ON s.id = a.slot_id
		WHERE ` + whereSQL + `
		ORDER BY a.created_at DESC
		LIMIT $` + string(rune('0'+argIdx)) + ` OFFSET $` + string(rune('0'+argIdx+1))
	args = append(args, limit, offset)

	rows, err := db.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var appts []models.Appointment
	for rows.Next() {
		var a models.Appointment
		var s models.ScheduleSlot
		err := rows.Scan(
			&a.ID, &a.SlotID, &a.PatientName, &a.PatientPhone, &a.Status, &a.AppointmentNo, &a.CreatedAt,
			&s.ScheduleDate, &s.StartTime, &s.EndTime,
		)
		if err != nil {
			return nil, 0, err
		}
		a.Slot = &s
		appts = append(appts, a)
	}
	return appts, total, rows.Err()
}

func (r *appointmentRepo) CancelBySlotID(ctx context.Context, tx *sql.Tx, slotID int64, operator, reason string) (int64, error) {
	result, err := tx.ExecContext(ctx, `
		UPDATE appointments
		SET status = 'suspended'
		WHERE slot_id = $1 AND status IN ('pending', 'confirmed')
	`, slotID)
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	return affected, nil
}

type appointmentLogRepo struct{}

func NewAppointmentLogRepository() AppointmentLogRepository {
	return &appointmentLogRepo{}
}

func (r *appointmentLogRepo) Create(ctx context.Context, log *models.AppointmentLog) error {
	return db.GetDB().QueryRowContext(ctx, `
		INSERT INTO appointment_logs (appointment_id, old_status, new_status, operator, reason)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`, log.AppointmentID, log.OldStatus, log.NewStatus, log.Operator, log.Reason,
	).Scan(&log.ID, &log.CreatedAt)
}

func (r *appointmentLogRepo) ListByAppointmentID(ctx context.Context, appointmentID int64) ([]models.AppointmentLog, error) {
	rows, err := db.GetDB().QueryContext(ctx, `
		SELECT id, appointment_id, old_status, new_status, operator, reason, created_at
		FROM appointment_logs
		WHERE appointment_id = $1
		ORDER BY created_at DESC
	`, appointmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.AppointmentLog
	for rows.Next() {
		var l models.AppointmentLog
		err := rows.Scan(&l.ID, &l.AppointmentID, &l.OldStatus, &l.NewStatus, &l.Operator, &l.Reason, &l.CreatedAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

type suspensionRepo struct{}

func NewSuspensionRepository() SuspensionRepository {
	return &suspensionRepo{}
}

func (r *suspensionRepo) Create(ctx context.Context, tx *sql.Tx, s *models.SuspensionDay) error {
	return tx.QueryRowContext(ctx, `
		INSERT INTO suspension_days (doctor_id, suspend_date, reason)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, s.DoctorID, s.SuspendDate, s.Reason,
	).Scan(&s.ID, &s.CreatedAt)
}

func (r *suspensionRepo) Exists(ctx context.Context, doctorID int64, date string) (bool, error) {
	var exists bool
	err := db.GetDB().QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM suspension_days WHERE doctor_id = $1 AND suspend_date = $2)
	`, doctorID, date).Scan(&exists)
	return exists, err
}
