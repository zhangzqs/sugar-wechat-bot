package zerologger

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/natefinch/lumberjack"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

type Config struct {
	LogFile    string `yaml:"log_file"`    // 日志文件名，默认输出到标准输出
	LogConsole bool   `yaml:"log_console"` // 是否同时输出到控制台
	Level      string `yaml:"level"`       // 日志级别
	MaxSize    int    `yaml:"max_size"`    // 单个日志文件的大小
	MaxBackups int    `yaml:"max_backups"` // 保留的日志文件个数
	MaxAge     int    `yaml:"max_age"`     // 日志保留的最长时间：天
	Compress   bool   `yaml:"compress"`    // 日志是否压缩
	LocalTime  bool   `yaml:"local_time"`  // 是否使用本地时间
}

func (cfg *Config) Validate() error {
	if cfg.Level == "" {
		cfg.Level = "info"
	}
	if cfg.MaxSize == 0 {
		cfg.MaxSize = 512
	}
	if cfg.MaxBackups == 0 {
		cfg.MaxBackups = 10
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 15
	}
	return nil
}

type Logger struct {
	*zerolog.Logger
	closer []io.Closer
}

func (l *Logger) Close() error {
	var errs []error
	for _, c := range l.closer {
		if err := c.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close logger: %w", err))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func init() {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
}

func MustNewLogger(cfg *Config) Logger {
	logger, err := NewLogger(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}
	return *logger
}

func NewLogger(cfg *Config) (*Logger, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid logger config: %w", err)
	}

	var writers []io.Writer
	var closer []io.Closer
	if cfg.LogFile != "" {
		// 确保日志文件夹已创建
		dir := filepath.Dir(cfg.LogFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory %s: %w", dir, err)
		}
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.LogFile,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			LocalTime:  cfg.LocalTime,
			Compress:   cfg.Compress,
		}
		writers = append(writers, fileWriter)
		closer = append(closer, fileWriter)
	}
	if cfg.LogConsole {
		consoleWriter := zerolog.NewConsoleWriter()
		writers = append(writers, consoleWriter)
		closer = append(closer, consoleWriter)
	}
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %s: %w", cfg.Level, err)
	}
	logger := zerolog.New(io.MultiWriter(writers...))
	logger = logger.With().Caller().Stack().Timestamp().Logger().Level(level)
	return &Logger{
		Logger: &logger,
		closer: closer,
	}, nil
}
