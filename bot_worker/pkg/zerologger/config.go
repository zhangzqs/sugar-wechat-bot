package zerologger

import (
	"fmt"

	"github.com/natefinch/lumberjack"
	"github.com/rs/zerolog"
)

type Config struct {
	LogFile    string `yaml:"log_file"`    // 日志文件名，默认输出到标准输出
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
	hook *lumberjack.Logger // 用于日志轮转
}

func (l *Logger) Close() error {
	if l.hook != nil {
		if err := l.hook.Close(); err != nil {
			return fmt.Errorf("failed to close logger hook: %w", err)
		}
		l.hook = nil
	}
	return nil
}

func NewLogger(cfg *Config) (*Logger, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid logger config: %w", err)
	}
	writer := &lumberjack.Logger{
		Filename:   cfg.LogFile,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		LocalTime:  cfg.LocalTime,
		Compress:   cfg.Compress,
	}

	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %s: %w", cfg.Level, err)
	}
	logger := zerolog.New(writer)
	logger = logger.With().Logger().Level(level)
	return &Logger{
		Logger: &logger,
		hook:   writer,
	}, nil
}
