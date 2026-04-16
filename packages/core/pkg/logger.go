package pkg

import (
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Wondermove-Inc/skuberplus/skuberplus-observability/packages/core/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logger *zap.Logger
	once   sync.Once
)

type zapWriter struct {
	logger *zap.Logger
}

func (w zapWriter) Write(p []byte) (n int, err error) {
	w.logger.Info(string(p))
	return len(p), nil
}

// NewZapWriter creates a new io.Writer that writes to a zap.Logger
func NewZapWriter(logger *zap.Logger) io.Writer {
	return zapWriter{logger: logger}
}

func CustomTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	// enc.AppendString(t.Format("2006-01-02T15:04:05"))
	enc.AppendString(t.Format(time.RFC3339))
}

func getEncoderConfig() zapcore.EncoderConfig {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		FunctionKey:    zapcore.OmitKey,                // 함수 이름도 포함
		EncodeLevel:    zapcore.CapitalLevelEncoder,    // Capitalize the log level names
		EncodeTime:     CustomTimeEncoder,              // Custom timestamp format
		EncodeDuration: zapcore.SecondsDurationEncoder, // Duration in seconds
		EncodeCaller:   zapcore.FullCallerEncoder,      // ShortCallerEncoder 대신 FullCallerEncoder 사용
	}
	return encoderConfig

}

func getRotateLogger(filename string) *lumberjack.Logger {
	// Set up lumberjack as a logger:
	rotateLogger := &lumberjack.Logger{
		Filename:   filename, // Or any other path
		MaxSize:    500,      // MB; after this size, a new log file is created
		MaxBackups: 3,        // Number of backups to keep
		MaxAge:     28,       // Days
		Compress:   true,     // Compress the backups using gzip
	}
	return rotateLogger
}

// InitLogger는 애플리케이션 시작 시 logger를 초기화합니다.
func InitLogger() {
	once.Do(func() {
		var err error
		logger, err = initializeLogger()
		if err != nil {
			log.Fatalf("Failed to initialize logger: %v", err)
		}
	})
}

func GetLogger() *zap.Logger {
	return logger
}

func initializeLogger() (*zap.Logger, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Configure the format
	encoderConfig := getEncoderConfig()

	// Create file and console encoders
	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)

	// Open the log file
	rotateLogger := getRotateLogger(cfg.Logging.File)

	// Create writers for file and console
	writeSyncer := zapcore.AddSync(rotateLogger)
	consoleWriter := zapcore.AddSync(os.Stdout)

	// Set the log level from config file
	defaultLogLevel, err := zapcore.ParseLevel(cfg.Logging.Level)
	if err != nil {
		log.Fatal("Cannot parse log level:", err)
	}

	// Create cores for writing to the file and console
	fileCore := zapcore.NewCore(fileEncoder, writeSyncer, defaultLogLevel)
	consoleCore := zapcore.NewCore(consoleEncoder, consoleWriter, defaultLogLevel)

	// Combine cores
	core := zapcore.NewTee(fileCore, consoleCore)

	// Create the logger with additional context information (caller, stack trace)
	// 에러 레벨일 때 스택트레이스를 포함하도록 설정
	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1), // 실제 호출 위치를 정확히 표시하기 위해 추가
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.Development(), // 개발 모드에서 더 자세한 스택트레이스 제공
	)

	return logger, nil
}
