package dto

import "clinic-appointment/internal/models"

type AppointmentCreateRequest struct {
	SlotID        int64  `json:"slot_id" binding:"required"`
	PatientName   string `json:"patient_name" binding:"required,max=100"`
	PatientPhone  string `json:"patient_phone" binding:"required,len=11"`
	PatientIDCard string `json:"patient_id_card" binding:"omitempty,len=18"`
}

type AppointmentCancelRequest struct {
	Reason string `json:"reason" binding:"max=200"`
}

type AppointmentListRequest struct {
	PatientPhone string `form:"patient_phone"`
	Status       string `form:"status"`
	Date         string `form:"date"`
	Page         int    `form:"page,default=1"`
	PageSize     int    `form:"page_size,default=20"`
}

type DoctorAppointmentListRequest struct {
	DoctorID int64  `form:"doctor_id" binding:"required"`
	Date     string `form:"date"`
	Status   string `form:"status"`
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
}

type ScheduleSlotGenerateRequest struct {
	DoctorID int64  `json:"doctor_id" binding:"required"`
	Date     string `json:"date" binding:"required"`
}

type ScheduleTemplateCreateRequest struct {
	DoctorID  int64  `json:"doctor_id" binding:"required"`
	DayOfWeek string `json:"day_of_week" binding:"required,oneof=monday tuesday wednesday thursday friday saturday sunday"`
	StartTime string `json:"start_time" binding:"required"`
	EndTime   string `json:"end_time" binding:"required"`
	Quota     int    `json:"quota" binding:"min=1"`
}

type ScheduleTemplateUpdateRequest struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Quota     int    `json:"quota" binding:"min=1"`
	IsActive  *bool  `json:"is_active"`
}

type ScheduleSlotUpdateRequest struct {
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	TotalQuota int    `json:"total_quota" binding:"min=1"`
}

type ScheduleConflictResponse struct {
	HasConflict bool                      `json:"has_conflict"`
	Conflicts   []models.ScheduleConflict `json:"conflicts"`
	Message     string                    `json:"message,omitempty"`
}

type ScheduleConflictCheckRequest struct {
	DoctorID     int64  `json:"doctor_id" binding:"required"`
	Date         string `json:"date" binding:"required"`
	StartTime    string `json:"start_time" binding:"required"`
	EndTime      string `json:"end_time" binding:"required"`
	ExcludeSlotID int64 `json:"exclude_slot_id"`
}

type SuspensionCreateRequest struct {
	DoctorID int64  `json:"doctor_id" binding:"required"`
	Date     string `json:"date" binding:"required"`
	Reason   string `json:"reason" binding:"max=200"`
}

type WaitlistJoinRequest struct {
	SlotID        int64  `json:"slot_id" binding:"required"`
	PatientName   string `json:"patient_name" binding:"required,max=100"`
	PatientPhone  string `json:"patient_phone" binding:"required,len=11"`
	PatientIDCard string `json:"patient_id_card" binding:"omitempty,len=18"`
}

type WaitlistCancelRequest struct {
	Reason string `json:"reason" binding:"max=200"`
}

type WaitlistListRequest struct {
	PatientPhone string `form:"patient_phone" binding:"required"`
	Status       string `form:"status"`
	Page         int    `form:"page,default=1"`
	PageSize     int    `form:"page_size,default=20"`
}

type PromoteResult struct {
	WaitlistID    int64  `json:"waitlist_id"`
	AppointmentID int64  `json:"appointment_id"`
	AppointmentNo string `json:"appointment_no"`
	PatientName   string `json:"patient_name"`
	PatientPhone  string `json:"patient_phone"`
	Position      int    `json:"position"`
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type PaginatedResponse struct {
	Total   int64       `json:"total"`
	Page    int         `json:"page"`
	Size    int         `json:"size"`
	Records interface{} `json:"records"`
}

func Success(data interface{}) Response {
	return Response{Code: 0, Message: "success", Data: data}
}

func Error(code int, message string) Response {
	return Response{Code: code, Message: message}
}

func Paginated(total int64, page, size int, records interface{}) Response {
	return Success(PaginatedResponse{
		Total:   total,
		Page:    page,
		Size:    size,
		Records: records,
	})
}
