package repository

import (
	"context"
	"database/sql"

	"clinic-appointment/internal/db"
	"clinic-appointment/internal/models"
)

type departmentRepo struct{}

func NewDepartmentRepository() DepartmentRepository {
	return &departmentRepo{}
}

func (r *departmentRepo) List(ctx context.Context) ([]models.Department, error) {
	rows, err := db.GetDB().QueryContext(ctx, `
		SELECT id, name, description, created_at, updated_at
		FROM departments ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var depts []models.Department
	for rows.Next() {
		var d models.Department
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		depts = append(depts, d)
	}
	return depts, rows.Err()
}

func (r *departmentRepo) GetByID(ctx context.Context, id int64) (*models.Department, error) {
	var d models.Department
	err := db.GetDB().QueryRowContext(ctx, `
		SELECT id, name, description, created_at, updated_at
		FROM departments WHERE id = $1
	`, id).Scan(&d.ID, &d.Name, &d.Description, &d.CreatedAt, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

type doctorRepo struct{}

func NewDoctorRepository() DoctorRepository {
	return &doctorRepo{}
}

func (r *doctorRepo) ListByDepartment(ctx context.Context, departmentID int64) ([]models.Doctor, error) {
	rows, err := db.GetDB().QueryContext(ctx, `
		SELECT d.id, d.department_id, d.name, d.title, d.description, d.created_at, d.updated_at,
		       dep.id, dep.name
		FROM doctors d
		LEFT JOIN departments dep ON dep.id = d.department_id
		WHERE d.department_id = $1
		ORDER BY d.id
	`, departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var doctors []models.Doctor
	for rows.Next() {
		var doc models.Doctor
		var dep models.Department
		err := rows.Scan(
			&doc.ID, &doc.DepartmentID, &doc.Name, &doc.Title, &doc.Description,
			&doc.CreatedAt, &doc.UpdatedAt,
			&dep.ID, &dep.Name,
		)
		if err != nil {
			return nil, err
		}
		doc.Department = &dep
		doctors = append(doctors, doc)
	}
	return doctors, rows.Err()
}

func (r *doctorRepo) GetByID(ctx context.Context, id int64) (*models.Doctor, error) {
	var doc models.Doctor
	var dep models.Department
	err := db.GetDB().QueryRowContext(ctx, `
		SELECT d.id, d.department_id, d.name, d.title, d.description, d.created_at, d.updated_at,
		       dep.id, dep.name
		FROM doctors d
		LEFT JOIN departments dep ON dep.id = d.department_id
		WHERE d.id = $1
	`, id).Scan(
		&doc.ID, &doc.DepartmentID, &doc.Name, &doc.Title, &doc.Description,
		&doc.CreatedAt, &doc.UpdatedAt,
		&dep.ID, &dep.Name,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	doc.Department = &dep
	return &doc, nil
}

type scheduleSlotRepo struct{}

func NewScheduleSlotRepository() ScheduleSlotRepository {
	return &scheduleSlotRepo{}
}

func (r *scheduleSlotRepo) ListByDoctorAndDateRange(ctx context.Context, doctorID int64, startDate, endDate string) ([]models.ScheduleSlot, error) {
	query := `
		SELECT s.id, s.doctor_id, s.schedule_date, s.start_time, s.end_time,
		       s.total_quota, s.used_quota, s.is_suspended, s.created_at, s.updated_at,
		       d.id, d.name, d.title
		FROM schedule_slots s
		LEFT JOIN doctors d ON d.id = s.doctor_id
		WHERE s.doctor_id = $1 AND s.schedule_date >= $2 AND s.schedule_date <= $3
		ORDER BY s.schedule_date, s.start_time
	`
	rows, err := db.GetDB().QueryContext(ctx, query, doctorID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []models.ScheduleSlot
	for rows.Next() {
		var slot models.ScheduleSlot
		var doc models.Doctor
		err := rows.Scan(
			&slot.ID, &slot.DoctorID, &slot.ScheduleDate, &slot.StartTime, &slot.EndTime,
			&slot.TotalQuota, &slot.UsedQuota, &slot.IsSuspended, &slot.CreatedAt, &slot.UpdatedAt,
			&doc.ID, &doc.Name, &doc.Title,
		)
		if err != nil {
			return nil, err
		}
		slot.Doctor = &doc
		slots = append(slots, slot)
	}
	return slots, rows.Err()
}

func (r *scheduleSlotRepo) GetByID(ctx context.Context, id int64) (*models.ScheduleSlot, error) {
	var slot models.ScheduleSlot
	var doc models.Doctor
	err := db.GetDB().QueryRowContext(ctx, `
		SELECT s.id, s.doctor_id, s.schedule_date, s.start_time, s.end_time,
		       s.total_quota, s.used_quota, s.is_suspended, s.created_at, s.updated_at,
		       d.id, d.name, d.title
		FROM schedule_slots s
		LEFT JOIN doctors d ON d.id = s.doctor_id
		WHERE s.id = $1
	`, id).Scan(
		&slot.ID, &slot.DoctorID, &slot.ScheduleDate, &slot.StartTime, &slot.EndTime,
		&slot.TotalQuota, &slot.UsedQuota, &slot.IsSuspended, &slot.CreatedAt, &slot.UpdatedAt,
		&doc.ID, &doc.Name, &doc.Title,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	slot.Doctor = &doc
	return &slot, nil
}

func (r *scheduleSlotRepo) GetByDoctorDateTime(ctx context.Context, doctorID int64, date, startTime, endTime string) (*models.ScheduleSlot, error) {
	var slot models.ScheduleSlot
	err := db.GetDB().QueryRowContext(ctx, `
		SELECT id, doctor_id, schedule_date, start_time, end_time,
		       total_quota, used_quota, is_suspended, created_at, updated_at
		FROM schedule_slots
		WHERE doctor_id = $1 AND schedule_date = $2 AND start_time = $3 AND end_time = $4
	`, doctorID, date, startTime, endTime).Scan(
		&slot.ID, &slot.DoctorID, &slot.ScheduleDate, &slot.StartTime, &slot.EndTime,
		&slot.TotalQuota, &slot.UsedQuota, &slot.IsSuspended, &slot.CreatedAt, &slot.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &slot, nil
}

func (r *scheduleSlotRepo) Create(ctx context.Context, slot *models.ScheduleSlot) error {
	return db.GetDB().QueryRowContext(ctx, `
		INSERT INTO schedule_slots (doctor_id, schedule_date, start_time, end_time, total_quota, used_quota)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`, slot.DoctorID, slot.ScheduleDate, slot.StartTime, slot.EndTime, slot.TotalQuota, slot.UsedQuota,
	).Scan(&slot.ID, &slot.CreatedAt, &slot.UpdatedAt)
}

func (r *scheduleSlotRepo) UpdateUsedQuota(ctx context.Context, tx *sql.Tx, slotID int64, delta int) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE schedule_slots
		SET used_quota = used_quota + $1
		WHERE id = $2
	`, delta, slotID)
	return err
}

func (r *scheduleSlotRepo) MarkSuspended(ctx context.Context, doctorID int64, date string) (int64, error) {
	result, err := db.GetDB().ExecContext(ctx, `
		UPDATE schedule_slots
		SET is_suspended = TRUE
		WHERE doctor_id = $1 AND schedule_date = $2
	`, doctorID, date)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *scheduleSlotRepo) Delete(ctx context.Context, id int64) error {
	_, err := db.GetDB().ExecContext(ctx, "DELETE FROM schedule_slots WHERE id = $1", id)
	return err
}
