package helloworld

import (
	"context"
	"errors"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/zhangzqs/sugar-wechat-bot/bot_runner_go/pkg/runner"
)

type Config struct {
	ListenAddr string `yaml:"listen_addr"` // HTTP 服务器地址
}

var _ runner.Runner = (*HelloWorldRunner)(nil)

type HelloWorldRunner struct {
	addr string
}

func New(cfg *Config) *HelloWorldRunner {
	if cfg.ListenAddr == "" {
		panic(errors.New("listen_addr is required"))
	}
	return &HelloWorldRunner{
		addr: cfg.ListenAddr,
	}
}

func (h *HelloWorldRunner) Name() string {
	return "HelloWorldRunner"
}

func (h *HelloWorldRunner) Run(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	logger.Info().Msg("Hello, World!")

	server := &http.Server{
		Addr: h.addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("Hello, World!"))
			if err != nil {
				logger.Error().Err(err).Msg("Failed to write response")
			}
		}),
	}

	go func() {
		logger.Info().Msg("HTTP server started on :8080")
		if err := server.ListenAndServe(); err != nil {
			logger.Error().Err(err).Msg("Failed to start HTTP server")
		}
	}()

	// 等待上下文取消
	<-ctx.Done()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("Failed to shutdown HTTP server")
	} else {
		logger.Info().Msg("HTTP server stopped successfully")
	}
	return nil
}
