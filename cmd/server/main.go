package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"clinic-appointment/internal/config"
	"clinic-appointment/internal/db"
	"clinic-appointment/internal/handler"
	"clinic-appointment/internal/logger"
	"clinic-appointment/internal/repository"
	"clinic-appointment/internal/server"
	"clinic-appointment/internal/service"

	"go.uber.org/zap"
)

var configPath = flag.String("config", "config/config.yaml", "path to config file")

func main() {
	flag.Parse()

	logger.Init()
	defer logger.Sync()

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("failed to load config", zap.Error(err))
		os.Exit(1)
	}

	if err := db.Init(cfg.Database); err != nil {
		logger.Error("failed to init database", zap.Error(err))
		os.Exit(1)
	}
	defer db.Close()

	deptRepo := repository.NewDepartmentRepository()
	doctorRepo := repository.NewDoctorRepository()
	slotRepo := repository.NewScheduleSlotRepository()
	apptRepo := repository.NewAppointmentRepository()
	logRepo := repository.NewAppointmentLogRepository()
	suspensionRepo := repository.NewSuspensionRepository()

	deptService := service.NewDepartmentService(deptRepo)
	doctorService := service.NewDoctorService(doctorRepo)
	scheduleService := service.NewScheduleService(slotRepo, suspensionRepo, doctorRepo, apptRepo, &cfg.Appointment)
	apptService := service.NewAppointmentService(apptRepo, logRepo, slotRepo, &cfg.Appointment)

	h := handler.NewHandler(deptService, doctorService, scheduleService, apptService)
	router := server.SetupRouter(h)

	addr := cfg.Server.Address()
	logger.Info("server starting", zap.String("addr", addr))

	go func() {
		if err := router.Run(addr); err != nil {
			logger.Error("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nserver shutting down gracefully")
}
