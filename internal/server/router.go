package server

import (
	"net/http"

	"clinic-appointment/internal/handler"

	"github.com/gin-gonic/gin"
)

func SetupRouter(h *handler.Handler) *gin.Engine {
	r := gin.Default()

	api := r.Group("/api/v1")
	{
		departments := api.Group("/departments")
		{
			departments.GET("", h.ListDepartments)
			departments.GET("/:id", h.GetDepartment)
			departments.GET("/:department_id/doctors", h.ListDoctors)
		}

		doctors := api.Group("/doctors")
		{
			doctors.GET("/:id", h.GetDoctor)
			doctors.POST("/slots/generate", h.GenerateScheduleSlots)
			doctors.GET("/:doctor_id/slots", h.ListScheduleSlots)
		}

		slots := api.Group("/slots")
		{
			slots.GET("/:id", h.GetScheduleSlot)
		}

		appointments := api.Group("/appointments")
		{
			appointments.POST("", h.CreateAppointment)
			appointments.GET("/:id", h.GetAppointment)
			appointments.GET("/no/:no", h.GetAppointmentByNo)
			appointments.POST("/:id/cancel", h.CancelAppointment)
			appointments.POST("/:id/confirm", h.ConfirmAppointment)
			appointments.POST("/:id/complete", h.CompleteAppointment)
			appointments.GET("/:id/logs", h.GetAppointmentLogs)
		}

		patient := api.Group("/patient")
		{
			patient.GET("/appointments", h.ListPatientAppointments)
		}

		doctor := api.Group("/doctor")
		{
			doctor.GET("/appointments", h.ListDoctorAppointments)
			doctor.POST("/suspend", h.CreateSuspension)
		}
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return r
}
