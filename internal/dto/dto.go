package dto

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

type SuspensionCreateRequest struct {
	DoctorID int64  `json:"doctor_id" binding:"required"`
	Date     string `json:"date" binding:"required"`
	Reason   string `json:"reason" binding:"max=200"`
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
