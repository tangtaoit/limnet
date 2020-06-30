package limlog

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var logger *zap.Logger
var errorLogger *zap.Logger
var warnLogger *zap.Logger

func init() {
	infoWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "info.log",
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
	})
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(newEncoderConfig()),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(infoWriter)),
		zap.DebugLevel,
	)
	logger = zap.New(core)

	errorWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "error.log",
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
	})
	core = zapcore.NewCore(
		zapcore.NewConsoleEncoder(newEncoderConfig()),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(errorWriter)),
		zap.ErrorLevel,
	)
	errorLogger = zap.New(core)

	warnWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "warn.log",
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
	})
	core = zapcore.NewCore(
		zapcore.NewConsoleEncoder(newEncoderConfig()),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(warnWriter)),
		zap.WarnLevel,
	)
	warnLogger = zap.New(core)

}

func newEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		// Keys can be anything except the empty string.
		TimeKey:        "T",
		LevelKey:       "L",
		NameKey:        "N",
		CallerKey:      "C",
		MessageKey:     "M",
		StacktraceKey:  "S",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     timeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}
func timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

// Info Info
func Info(msg string, fields ...zap.Field) {
	logger.Info(msg, fields...)
}

// Debug Debug
func Debug(msg string, fields ...zap.Field) {
	logger.Debug(msg, fields...)

}

// Error Error
func Error(msg string, fields ...zap.Field) {
	errorLogger.Error(msg, fields...)
}

// Warn Warn
func Warn(msg string, fields ...zap.Field) {
	warnLogger.Warn(msg, fields...)
}

// Log Log
type Log interface {
	Info(msg string, fields ...zap.Field)
	Debug(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
}

// LIMLog TLog
type LIMLog struct {
	prefix string // 日志前缀
}

// NewLIMLog NewLIMLog
func NewLIMLog(prefix string) *LIMLog {

	return &LIMLog{prefix: prefix}
}

// Info Info
func (t *LIMLog) Info(msg string, fields ...zap.Field) {
	Info(fmt.Sprintf("【%s】%s", t.prefix, msg), fields...)
}

// Debug Debug
func (t *LIMLog) Debug(msg string, fields ...zap.Field) {
	Debug(fmt.Sprintf("【%s】%s", t.prefix, msg), fields...)
}

// Error Error
func (t *LIMLog) Error(msg string, fields ...zap.Field) {
	Error(fmt.Sprintf("【%s】%s", t.prefix, msg), fields...)
}

// Warn Warn
func (t *LIMLog) Warn(msg string, fields ...zap.Field) {
	Warn(fmt.Sprintf("【%s】%s", t.prefix, msg), fields...)
}
