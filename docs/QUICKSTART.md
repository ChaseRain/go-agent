# 🚀 Go Agent 快速开始

## 环境准备

### 1. 安装Go语言
确保安装了Go 1.21或更高版本：
```bash
# 检查Go版本
go version

# 如果未安装，请访问 https://golang.org/dl/ 下载安装
```

### 2. 获取OpenAI API密钥
1. 访问 [OpenAI Platform](https://platform.openai.com/)
2. 注册并获取API密钥
3. 设置环境变量：
```bash
export OPENAI_API_KEY="your-api-key-here"
```

## 快速安装

### 方法1：使用设置脚本（推荐）
```bash
cd go-agent
./scripts/setup.sh
```

### 方法2：手动设置
```bash
cd go-agent

# 初始化依赖
go mod tidy

# 创建目录
mkdir -p bin examples/output data

# 构建项目
go build -o bin/go-agent ./cmd/main.go
```

## 🎯 第一次运行

### 1. 查看配置
```bash
make config
# 或
go run cmd/main.go config
```

### 2. 运行简单示例
```bash
make example
# 或
go run examples/simple_agent.go
```

### 3. 运行工具示例
```bash
make example-tools
# 或
go run examples/with_tools.go
```

### 4. 交互模式
```bash
make run
# 或
go run cmd/main.go run
```

## 📋 示例任务

在交互模式中，您可以尝试以下任务：

### 基础任务
```
> 解释什么是Go语言的goroutine
> 比较Go和Python的优缺点
> 制定一个Go语言学习计划
```

### 计算任务
```
> 计算1到100的和
> 解释斐波那契数列的算法
> 帮我设计一个简单的计算器程序
```

### 文本处理
```
> 分析这段文字的特点：Go is a programming language
> 帮我写一个产品介绍
> 总结Go语言的主要特性
```

## 🔧 自定义配置

### 创建配置文件
```bash
# 创建配置文件
cat > config.json << EOF
{
  "llm_config": {
    "provider": "openai",
    "model": "gpt-3.5-turbo",
    "api_key": "your-api-key",
    "temperature": 0.7,
    "max_tokens": 2048
  },
  "execution_config": {
    "max_round": 20,
    "parallel": false,
    "stream": true,
    "save_dir": "./data"
  },
  "agent_config": {
    "name": "MyAgent",
    "role_description": "我的专属助手",
    "role_prompt": "你是我的专属助手，帮助我学习Go语言。"
  }
}
EOF
```

### 使用配置文件
```go
// 在代码中加载配置
cfg := config.NewConfigManager()
err := cfg.LoadFromFile("config.json")
```

## 🛠️ 添加自定义工具

### 1. 创建工具文件
```go
// pkg/tools/my_tool.go
package tools

import "fmt"

type MyTool struct {
    Name        string
    Description string
}

func NewMyTool() *MyTool {
    return &MyTool{
        Name:        "my_tool",
        Description: "我的自定义工具",
    }
}

func (mt *MyTool) Execute(input string) (string, error) {
    return fmt.Sprintf("处理输入: %s", input), nil
}
```

### 2. 注册工具
```go
// 在main.go中注册
dynAgent.RegisterFunction("my_tool", tools.NewMyTool())
```

## 🐛 常见问题

### 1. API密钥问题
```bash
# 错误: 请设置 OPENAI_API_KEY 环境变量
export OPENAI_API_KEY="sk-your-key-here"

# 验证设置
echo $OPENAI_API_KEY
```

### 2. 编译错误
```bash
# 清理并重新构建
make clean
make build

# 检查依赖
go mod tidy
```

### 3. 运行时错误
```bash
# 检查日志
ls -la data/
cat logs/agent.log

# 清理状态
rm -rf data/save_agent/*
```

### 4. 网络问题
```bash
# 设置代理（如果需要）
export HTTPS_PROXY=http://proxy.example.com:8080
export HTTP_PROXY=http://proxy.example.com:8080

# 使用自定义API端点
export OPENAI_BASE_URL="https://your-custom-endpoint.com/v1"
```

## 📈 性能优化

### 1. 并行执行
```go
cfg.ExecutionConfig.Parallel = true
```

### 2. 调整Token限制
```go
cfg.LLMConfig.MaxTokens = 4096
```

### 3. 使用更强的模型
```go
cfg.LLMConfig.Model = "gpt-4"
```

## 🔍 调试技巧

### 1. 启用调试模式
```bash
export GO_AGENT_DEBUG=true
go run cmd/main.go run
```

### 2. 查看状态文件
```bash
# 查看保存的状态
find data/save_agent -name "*.json" -exec cat {} \;
```

### 3. 分析执行流程
```bash
# 查看生成的文件
ls -la examples/output/
cat examples/output/*.md
```

## 🎓 学习建议

### 第1天：基础了解
1. 阅读 `README.md`
2. 运行 `make example`
3. 查看 `internal/models/models.go`

### 第2-3天：核心组件
1. 学习配置系统：`internal/config/config.go`
2. 理解消息管理：`internal/messaging/message_manager.go`
3. 运行 `make example-tools`

### 第4-7天：进阶功能
1. 任务规划：`internal/planning/task_planner.go`
2. 任务执行：`internal/execution/task_executor.go`
3. 状态管理：`internal/state/state_manager.go`

### 第2周：深入理解
1. 主控制器：`internal/agent/dynagent.go`
2. LLM集成：`internal/llm/call_llm.go`
3. 结果处理：`internal/results/result_processor.go`

### 第3周：实践项目
1. 添加自定义工具
2. 扩展LLM提供商
3. 实现Web API接口
4. 添加数据库支持

## 🎉 恭喜！

您现在已经准备好开始Go Agent的学习之旅了！

记住：
- 从简单的示例开始
- 逐步理解每个组件
- 多实践，多思考
- 参考文档和源码注释

祝您学习愉快！🚀
