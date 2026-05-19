package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"clinic-appointment/internal/dto"
	"clinic-appointment/internal/models"
	"clinic-appointment/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

type Handler struct {
	deptService        *service.DepartmentService
	doctorService      *service.DoctorService
	scheduleService    *service.ScheduleService
	appointmentService *service.AppointmentService
}

func NewHandler(
	dept *service.DepartmentService,
	doctor *service.DoctorService,
	schedule *service.ScheduleService,
	appt *service.AppointmentService,
) *Handler {
	return &Handler{
		deptService:        dept,
		doctorService:      doctor,
		scheduleService:    schedule,
		appointmentService: appt,
	}
}

func (h *Handler) ctx(c *gin.Context) context.Context {
	return c.Request.Context()
}

func (h *Handler) bindAndValidate(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBind(req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(400, "invalid request: "+err.Error()))
		return false
	}
	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(400, "validation failed: "+err.Error()))
		return false
	}
	return true
}

func (h *Handler) handleError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}

	switch err {
	case service.ErrDepartmentNotFound,
		service.ErrDoctorNotFound,
		service.ErrSlotNotFound,
		service.ErrAppointmentNotFound:
		c.JSON(http.StatusNotFound, dto.Error(404, err.Error()))
	case service.ErrSlotSuspended,
		service.ErrSlotNoQuota,
		service.ErrInvalidDate,
		service.ErrDateInPast,
		service.ErrSuspensionExists,
		service.ErrInvalidStatus,
		service.ErrCannotCancelAfterStart,
		service.ErrCannotCancelPastWindow,
		service.ErrAlreadyCancelled,
		service.ErrDuplicateAppointment,
		service.ErrSlotAlreadyExists:
		c.JSON(http.StatusBadRequest, dto.Error(400, err.Error()))
	default:
		c.JSON(http.StatusInternalServerError, dto.Error(500, "internal error: "+err.Error()))
	}
	return true
}

func (h *Handler) ListDepartments(c *gin.Context) {
	depts, err := h.deptService.ListDepartments(h.ctx(c))
	if h.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto.Success(depts))
}

func (h *Handler) GetDepartment(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(400, "invalid id"))
		return
	}
	dept, err := h.deptService.GetDepartment(h.ctx(c), id)
	if h.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto.Success(dept))
}

func (h *Handler) ListDoctors(c *gin.Context) {
	deptID, err := strconv.ParseInt(c.Param("department_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(400, "invalid department id"))
		return
	}
	doctors, err := h.doctorService.ListDoctorsByDepartment(h.ctx(c), deptID)
	if h.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto.Success(doctors))
}

func (h *Handler) GetDoctor(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(400, "invalid id"))
		return
	}
	doctor, err := h.doctorService.GetDoctor(h.ctx(c), id)
	if h.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto.Success(doctor))
}

func (h *Handler) ListScheduleSlots(c *gin.Context) {
	doctorID, err := strconv.ParseInt(c.Param("doctor_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(400, "invalid doctor id"))
		return
	}
	startDate := c.DefaultQuery("start_date", time.Now().Format("2006-01-02"))
	endDate := c.DefaultQuery("end_date", time.Now().AddDate(0, 0, 7).Format("2006-01-02"))

	slots, err := h.scheduleService.ListSlots(h.ctx(c), doctorID, startDate, endDate)
	if h.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto.Success(slots))
}

func (h *Handler) GetScheduleSlot(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(400, "invalid id"))
		return
	}
	slot, err := h.scheduleService.GetSlot(h.ctx(c), id)
	if h.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto.Success(slot))
}

func (h *Handler) GenerateScheduleSlots(c *gin.Context) {
	var req dto.ScheduleSlotGenerateRequest
	if !h.bindAndValidate(c, &req) {
		return
	}
	err := h.scheduleService.GenerateSlotsForDate(h.ctx(c), req.DoctorID, req.Date)
	if h.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto.Success(nil))
}

func (h *Handler) CreateAppointment(c *gin.Context) {
	var req dto.AppointmentCreateRequest
	if !h.bindAndValidate(c, &req) {
		return
	}
	appt, err := h.appointmentService.CreateAppointment(h.ctx(c), req.SlotID, req.PatientName, req.PatientPhone, req.PatientIDCard)
	if h.handleError(c, err) {
		return
	}
	c.JSON(http.StatusCreated, dto.Success(appt))
}

func (h *Handler) GetAppointment(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(400, "invalid id"))
		return
	}
	appt, err := h.appointmentService.GetAppointment(h.ctx(c), id)
	if h.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto.Success(appt))
}

func (h *Handler) GetAppointmentByNo(c *gin.Context) {
	no := c.Param("no")
	appt, err := h.appointmentService.GetAppointmentByNo(h.ctx(c), no)
	if h.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto.Success(appt))
}

func (h *Handler) CancelAppointment(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(400, "invalid id"))
		return
	}
	var req dto.AppointmentCancelRequest
	if !h.bindAndValidate(c, &req) {
		return
	}
	err = h.appointmentService.CancelAppointment(h.ctx(c), id, req.Reason)
	if h.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto.Success(nil))
}

func (h *Handler) ListPatientAppointments(c *gin.Context) {
	var req dto.AppointmentListRequest
	if !h.bindAndValidate(c, &req) {
		return
	}
	appts, total, err := h.appointmentService.ListByPatient(
		h.ctx(c),
		req.PatientPhone,
		models.AppointmentStatus(req.Status),
		req.Date,
		req.Page,
		req.PageSize,
	)
	if h.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto.Paginated(total, req.Page, req.PageSize, appts))
}

func (h *Handler) ListDoctorAppointments(c *gin.Context) {
	var req dto.DoctorAppointmentListRequest
	if !h.bindAndValidate(c, &req) {
		return
	}
	appts, total, err := h.appointmentService.ListByDoctor(
		h.ctx(c),
		req.DoctorID,
		req.Date,
		models.AppointmentStatus(req.Status),
		req.Page,
		req.PageSize,
	)
	if h.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto.Paginated(total, req.Page, req.PageSize, appts))
}

func (h *Handler) ConfirmAppointment(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(400, "invalid id"))
		return
	}
	err = h.appointmentService.ConfirmAppointment(h.ctx(c), id)
	if h.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto.Success(nil))
}

func (h *Handler) CompleteAppointment(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(400, "invalid id"))
		return
	}
	err = h.appointmentService.CompleteAppointment(h.ctx(c), id)
	if h.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto.Success(nil))
}

func (h *Handler) CreateSuspension(c *gin.Context) {
	var req dto.SuspensionCreateRequest
	if !h.bindAndValidate(c, &req) {
		return
	}
	cancelled, err := h.scheduleService.CreateSuspension(h.ctx(c), req.DoctorID, req.Date, req.Reason)
	if h.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto.Success(gin.H{
		"cancelled_appointments": cancelled,
	}))
}

func (h *Handler) GetAppointmentLogs(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error(400, "invalid id"))
		return
	}
	logs, err := h.appointmentService.GetAppointmentLogs(h.ctx(c), id)
	if h.handleError(c, err) {
		return
	}
	c.JSON(http.StatusOK, dto.Success(logs))
}
