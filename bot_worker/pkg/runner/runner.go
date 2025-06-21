package runner

import (
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/rs/zerolog"
)

type Runner interface {
	io.Closer
	Run() error
	Name() string
}

func Run(svrs ...Runner) {
	logger := zerolog.DefaultContextLogger
	if logger == nil {
		logger1 := zerolog.New(os.Stdout).With().Timestamp().Logger().Level(zerolog.InfoLevel)
		logger = &logger1
	}
	{ // 只统计非 nil 服务
		var realSvrs []Runner
		for _, svr := range svrs {
			if svr != nil {
				realSvrs = append(realSvrs, svr)
			} else {
				logger.Error().Msg("server is nil, skipping")
			}
		}
		svrs = realSvrs
	}

	var wg sync.WaitGroup
	for _, svr := range svrs {
		wg.Add(1)
		go func(s Runner) {
			defer wg.Done()
			logger.Info().Str("service", s.Name()).Msg("started")
			if err := s.Run(); err != nil {
				logger.Error().Err(err).Str("service", svr.Name()).Msg("stopped due to error")
			} else {
				logger.Info().Str("service", svr.Name()).Msg("stopped successfully")
			}
		}(svr)
	}

	// 等待所有服务运行完毕
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGABRT, syscall.SIGSEGV)
	signal.Ignore(syscall.SIGPIPE, syscall.SIGHUP)
	logger.Info().Str("signal", (<-signalCh).String()).Msg("received signal, stopping")
	for _, svr := range svrs {
		if err := svr.Close(); err != nil {
			logger.Error().Err(err).Str("service", svr.Name()).Msg("failed to close server")
		} else {
			logger.Info().Str("service", svr.Name()).Msg("server closed successfully")
		}
	}
	wg.Wait()
	logger.Info().Msg("all servers have stopped")
}
