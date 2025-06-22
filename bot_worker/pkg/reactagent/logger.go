package reactagent

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/rs/zerolog"
)

type LoggerCallback struct {
	callbacks.HandlerBuilder
}

// 从 context 中提取 zerolog.Logger，若没有则返回默认 logger
func getLogger(ctx context.Context) zerolog.Logger {
	logger := zerolog.Ctx(ctx)
	if logger == nil {
		l := zerolog.New(io.Discard)
		return l
	}
	return *logger
}

func (cb *LoggerCallback) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	logger := getLogger(ctx)
	inputStr, _ := json.MarshalIndent(input, "", "  ")
	logger.Info().Str("event", "OnStart").RawJSON("input", inputStr).Msg("start callback")
	return ctx
}

func (cb *LoggerCallback) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	logger := getLogger(ctx)
	outputStr, _ := json.MarshalIndent(output, "", "  ")
	logger.Info().Str("event", "OnEnd").RawJSON("output", outputStr).Msg("end callback")
	return ctx
}

func (cb *LoggerCallback) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	logger := getLogger(ctx)
	logger.Error().Str("event", "OnError").Err(err).Msg("callback error")
	return ctx
}

func (cb *LoggerCallback) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo,
	output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {

	var graphInfoName = react.GraphName
	logger := getLogger(ctx)

	go func() {
		defer func() {
			if err := recover(); err != nil {
				logger.Error().Interface("panic", err).Msg("[OnEndStream] panic")
			}
		}()

		defer output.Close()

		logger.Info().Str("event", "OnEndStream").Msg("stream output started")
		for {
			frame, err := output.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				logger.Error().Err(err).Msg("internal error in stream output")
				return
			}

			s, err := json.Marshal(frame)
			if err != nil {
				logger.Error().Err(err).Msg("marshal frame error")
				return
			}

			if info.Name == graphInfoName {
				logger.Info().Str("service", info.Name).RawJSON("frame", s).Msg("stream frame")
			}
		}

	}()
	return ctx
}

func (cb *LoggerCallback) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo,
	input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	defer input.Close()
	return ctx
}
