package helloworld

import (
	"errors"
	"net/http"
	"strings"

	"github.com/zhangzqs/sugar-wechat-bot/bot_worker/pkg/runner"
	"github.com/zhangzqs/sugar-wechat-bot/bot_worker/pkg/zerologger"
)

type Config struct {
	Logger     zerologger.Config `yaml:"logger"`      // 日志配置
	ListenAddr string            `yaml:"listen_addr"` // HTTP 服务器地址
}

var _ runner.Runner = (*HelloWorldRunner)(nil)

type HelloWorldRunner struct {
	logger *zerologger.Logger
	addr   string
	server *http.Server
}

func NewHelloWorldRunner(cfg *Config) (*HelloWorldRunner, error) {
	if cfg.ListenAddr == "" {
		return nil, errors.New("listen_addr must be set")
	}
	if err := cfg.Logger.Validate(); err != nil {
		return nil, err
	}

	logger, err := zerologger.NewLogger(&cfg.Logger)
	if err != nil {
		return nil, err
	}

	return &HelloWorldRunner{
		logger: logger,
		addr:   cfg.ListenAddr,
	}, nil
}

func (h *HelloWorldRunner) Close() error {
	if h.server != nil {
		if err := h.server.Close(); err != nil {
			h.logger.Error().Err(err).Msg("Failed to close HTTP server")
		}
		h.server = nil
	}
	if h.logger != nil {
		if err := h.logger.Close(); err != nil {
			if !strings.Contains(err.Error(), "already closed") {
				panic(err)
			}
		}
		h.logger = nil
	}
	return nil
}

func (h *HelloWorldRunner) Name() string {
	return "HelloWorldRunner"
}

func (h *HelloWorldRunner) Run() error {
	h.logger.Info().Msg("Hello, World!")

	server := &http.Server{
		Addr: h.addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("Hello, World!"))
			if err != nil {
				h.logger.Error().Err(err).Msg("Failed to write response")
			}
		}),
	}
	h.server = server

	if err := server.ListenAndServe(); err != nil {
		h.logger.Error().Err(err).Msg("Failed to start HTTP server")
		return err
	}
	h.logger.Info().Msg("HTTP server started on :8080")
	return nil
}
