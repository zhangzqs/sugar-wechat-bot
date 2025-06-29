package runner

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/rs/zerolog"
)

type Runner interface {
	Name() string
	Run(ctx context.Context) error
}

func Run(logger zerolog.Logger, svrs ...Runner) {
	for _, svr := range svrs {
		if svr == nil {
			panic("runner cannot be nil")
		}
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for _, svr := range svrs {
		wg.Add(1)
		go func(s Runner, logger zerolog.Logger) {
			defer wg.Done()
			logger = logger.With().Str("service", s.Name()).Logger()
			ctx := logger.WithContext(ctx)

			if err := s.Run(ctx); err != nil {
				logger.Error().Err(err).Msg("stopped due to error")
			} else {
				logger.Info().Msg("stopped successfully")
			}
		}(svr, logger)
	}

	// 等待所有服务运行完毕
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGABRT, syscall.SIGSEGV)
	signal.Ignore(syscall.SIGPIPE, syscall.SIGHUP)
	logger.Info().Str("signal", (<-signalCh).String()).Msg("received signal, stopping")

	// 发送停止信号给所有服务
	cancelFunc()

	// 等待所有服务停止
	wg.Wait()

	logger.Info().Msg("all servers have stopped")
}
