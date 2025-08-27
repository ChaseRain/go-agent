#!/bin/bash

# Go Agent 设置脚本

set -e

echo "🚀 Go Agent 项目设置"
echo "===================="

# 检查Go版本
echo "🔍 检查Go版本..."
if ! command -v go &> /dev/null; then
    echo "❌ Go未安装，请先安装Go 1.21或更高版本"
    echo "   下载地址: https://golang.org/dl/"
    exit 1
fi

GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
echo "✅ Go版本: $GO_VERSION"

# 检查Go版本是否满足要求
REQUIRED_VERSION="1.21"
if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo "❌ Go版本过低，需要1.21或更高版本"
    exit 1
fi

# 初始化Go模块
echo "📦 初始化Go模块..."
if [ ! -f "go.mod" ]; then
    go mod init go-agent
fi

# 下载依赖
echo "📥 下载依赖..."
go mod tidy

# 创建必要的目录
echo "📁 创建目录结构..."
mkdir -p bin
mkdir -p examples/output
mkdir -p data/save_agent
mkdir -p logs

# 检查API密钥
echo "🔑 检查API密钥..."
if [ -z "$OPENAI_API_KEY" ]; then
    echo "⚠️  OPENAI_API_KEY 未设置"
    echo "   请设置环境变量: export OPENAI_API_KEY='your-api-key'"
    echo "   或创建 .env 文件"
    
    # 创建示例.env文件
    if [ ! -f ".env" ]; then
        cat > .env << EOF
# Go Agent 环境配置
OPENAI_API_KEY=your-openai-api-key-here
OPENAI_BASE_URL=https://api.openai.com/v1

# 可选配置
GO_AGENT_DEBUG=false
GO_AGENT_LOG_LEVEL=info
EOF
        echo "📝 已创建 .env 示例文件，请填入您的API密钥"
    fi
else
    echo "✅ OPENAI_API_KEY 已设置"
fi

# 构建项目
echo "🔨 构建项目..."
go build -o bin/go-agent ./cmd/main.go

# 运行基本测试
echo "🧪 运行基本测试..."
if go test -v ./internal/models/ > /dev/null 2>&1; then
    echo "✅ 基本测试通过"
else
    echo "⚠️  基本测试未通过，但不影响使用"
fi

echo ""
echo "🎉 设置完成!"
echo ""
echo "📋 下一步:"
echo "   1. 设置API密钥: export OPENAI_API_KEY='your-key'"
echo "   2. 运行示例: make example"
echo "   3. 交互模式: make run"
echo "   4. 查看帮助: make help"
echo ""
echo "📚 学习资源:"
echo "   - 学习指南: docs/LEARNING_GUIDE.md"
echo "   - 架构文档: docs/ARCHITECTURE.md"
echo "   - 项目说明: README.md"
echo ""
echo "🚀 开始您的Go Agent之旅!"
