package service

import (
	"context"
	"errors"

	"clinic-appointment/internal/models"
	"clinic-appointment/internal/repository"
)

var (
	ErrDepartmentNotFound = errors.New("department not found")
	ErrDoctorNotFound     = errors.New("doctor not found")
)

type DepartmentService struct {
	departmentRepo repository.DepartmentRepository
}

func NewDepartmentService(dr repository.DepartmentRepository) *DepartmentService {
	return &DepartmentService{departmentRepo: dr}
}

func (s *DepartmentService) ListDepartments(ctx context.Context) ([]models.Department, error) {
	return s.departmentRepo.List(ctx)
}

func (s *DepartmentService) GetDepartment(ctx context.Context, id int64) (*models.Department, error) {
	dept, err := s.departmentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dept == nil {
		return nil, ErrDepartmentNotFound
	}
	return dept, nil
}

type DoctorService struct {
	doctorRepo repository.DoctorRepository
}

func NewDoctorService(dr repository.DoctorRepository) *DoctorService {
	return &DoctorService{doctorRepo: dr}
}

func (s *DoctorService) ListDoctorsByDepartment(ctx context.Context, departmentID int64) ([]models.Doctor, error) {
	return s.doctorRepo.ListByDepartment(ctx, departmentID)
}

func (s *DoctorService) GetDoctor(ctx context.Context, id int64) (*models.Doctor, error) {
	doc, err := s.doctorRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, ErrDoctorNotFound
	}
	return doc, nil
}
