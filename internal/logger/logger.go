package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

func Init() {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		zap.InfoLevel,
	)

	log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
}

func L() *zap.Logger {
	if log == nil {
		Init()
	}
	return log
}

func Sync() {
	if log != nil {
		_ = log.Sync()
	}
}

func Info(msg string, fields ...zap.Field) {
	L().Info(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	L().Error(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	L().Warn(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	L().Debug(msg, fields...)
}
