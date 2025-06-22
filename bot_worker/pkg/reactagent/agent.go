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
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rs/zerolog"
)

type ReactAgent struct {
	agent *react.Agent
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
		cli, err := client.NewSSEMCPClient(mcpServerCfg.BaseURL)
		if err != nil {
			logger.Error().Err(err).Str("mcp_server", mcpServerCfg.Name).Msg("failed to create MCP client")
			return nil, err
		}
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

	allTools = append(allTools, &ListTodoTool{})

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
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
		agent: agent,
	}
	return
}

func (r *ReactAgent) Question(ctx context.Context, question string) (string, error) {
	logger := zerolog.Ctx(ctx).With().Str("component", "reactagent").Logger()
	logger.Info().Str("question", question).Msg("Processing question")

	// 使用 ReactAgent 处理用户问题
	answer, err := r.agent.Generate(ctx, []*schema.Message{
		{
			Role:    schema.User,
			Content: question,
		},
	}, agent.WithComposeOptions(compose.WithCallbacks(&LoggerCallback{})))

	if err != nil {
		logger.Error().Err(err).Msg("Failed to process question")
		return "", err
	}

	logger.Info().Str("answer", answer.String()).Msg("Question processed successfully")
	return answer.String(), nil
}

type ListTodoTool struct{}

func (lt *ListTodoTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "list_todo",
		Desc: "List all todo items",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"finished": {
				Desc:     "filter todo items if finished",
				Type:     schema.Boolean,
				Required: false,
			},
		}),
	}, nil
}

func (lt *ListTodoTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// Mock调用逻辑
	return `{"todos": [{"id": "1", "content": "在2024年12月10日之前完成Eino项目演示文稿的准备工作", "started_at": 1717401600, "deadline": 1717488000, "done": false}]}`, nil
}
