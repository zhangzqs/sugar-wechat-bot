package reactagent

type Config struct {
	SystemPrompt string      `yaml:"system_prompt"` // 系统提示词
	Model        ModelConfig `yaml:"model"`         // 模型配置
	MCPTools     []MCPServer `yaml:"mcp_tools"`     // MCP 工具配置
}

type MCPServer struct {
	Name         string   `yaml:"name"`           // MCP 服务器名称
	Version      string   `yaml:"version"`        // MCP 服务器版本
	BaseURL      string   `yaml:"base_url"`       // MCP 服务器基础 URL
	ToolNameList []string `yaml:"tool_name_list"` // 过滤所需 MCP 工具名称列表
}

type ModelConfig struct {
	BaseURL string `yaml:"base_url"` // 基础 URL
	APIKey  string `yaml:"api_key"`  // API Key
	Model   string `yaml:"model"`    // 模型名称
}
