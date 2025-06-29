package reactagent

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/openai"
	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rs/zerolog"
)

type ReactAgent struct {
	agent        *react.Agent
	systemPrompt string
}

func New(ctx context.Context, cfg *Config) (ret *ReactAgent, err error) {
	logger := zerolog.Ctx(ctx).With().Str("component", "reactagent").Logger()
	// 初始化LLM
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: cfg.Model.BaseURL,
		APIKey:  cfg.Model.APIKey,
		Model:   cfg.Model.Model,
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to initialize chat model")
		return nil, err
	}

	var allTools []tool.BaseTool

	// 初始化所有MCP工具
	for _, mcpServerCfg := range cfg.MCPTools {
		tp, err := transport.NewSSE(mcpServerCfg.BaseURL)
		if err != nil {
			logger.Error().Err(err).Str("mcp_server", mcpServerCfg.Name).
				Msg("failed to create SSE transport for MCP server")
			return nil, err
		}
		if err := tp.Start(ctx); err != nil {
			logger.Error().Err(err).Str("mcp_server", mcpServerCfg.Name).
				Msg("failed to start SSE transport for MCP server")
			return nil, err
		}
		cli := client.NewClient(tp)
		// 初始化MCP请求
		initRequest := mcp.InitializeRequest{
			Params: mcp.InitializeParams{
				ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
				ClientInfo: mcp.Implementation{
					Name:    mcpServerCfg.Name,
					Version: mcpServerCfg.Version,
				},
			},
		}
		_, err = cli.Initialize(ctx, initRequest)
		if err != nil {
			logger.Error().Err(err).Str("mcp_server", mcpServerCfg.Name).Msg("failed to initialize MCP client")
			return nil, err
		}
		mcpTools, err := einomcp.GetTools(ctx, &einomcp.Config{
			Cli:          cli,
			ToolNameList: mcpServerCfg.ToolNameList,
		})
		if err != nil {
			logger.Error().Err(err).Str("mcp_server", mcpServerCfg.Name).Msg("failed to get MCP tools")
			return nil, err
		}
		allTools = append(allTools, mcpTools...)
	}

	if len(allTools) == 0 {
		// 如果没有MCP工具，则添加一个占位符工具
		logger.Warn().Msg("No MCP tools found, using placeholder tool")
		allTools = append(allTools, &PlaceHolderTool{})
	}

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		MaxStep:          10,
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: allTools,
		},
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to create ReactAgent")
		return nil, err
	}

	ret = &ReactAgent{
		agent:        agent,
		systemPrompt: cfg.SystemPrompt,
	}
	return
}

func (r *ReactAgent) Question(ctx context.Context, question string) (string, error) {
	logger := zerolog.Ctx(ctx).With().Str("component", "reactagent").Logger()
	logger.Info().Str("question", question).Msg("Processing question")

	// 使用 ReactAgent 处理用户问题
	answer, err := r.agent.Generate(ctx, []*schema.Message{
		schema.SystemMessage(r.systemPrompt),
		schema.UserMessage(question),
	}, agent.WithComposeOptions(compose.WithCallbacks(&LoggerCallback{})))

	if err != nil {
		logger.Error().Err(err).Msg("Failed to process question")
		return "", err
	}

	logger.Info().Str("answer", answer.Content).Msg("Question processed successfully")
	return answer.Content, nil
}

type PlaceHolderTool struct{}

func (lt *PlaceHolderTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "PlaceHolderTool",
		Desc: "这仅是一个占位符工具，请不要调用它",
	}, nil
}

func (lt *PlaceHolderTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return "", nil
}
