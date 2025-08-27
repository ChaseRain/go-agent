// Package main 提供了Go智能代理框架的命令行接口
// 支持交互式和非交互式两种模式进行任务处理
package main

import (
	// 标准库包
	"bufio"   // 缓冲I/O操作
	"context" // 上下文管理
	"flag"    // 命令行参数解析
	"fmt"     // 格式化输入输出
	"log"     // 日志记录
	"os"      // 操作系统接口
	"strings" // 字符串操作

	// 本地包
	"go-agent/pkg/agent"  // 代理协调器
	"go-agent/pkg/config" // 配置管理
	"go-agent/pkg/models" // 数据模型和结构体
)

// main 是Go智能代理框架CLI的入口点
// 初始化代理系统并运行交互式或单查询模式
func main() {
	// 命令行参数定义
	configFile := flag.String("config", "config.yaml", "配置文件路径")
	interactive := flag.Bool("i", true, "交互模式")
	query := flag.String("q", "", "要处理的查询（非交互模式）")
	flag.Parse() // 解析命令行参数

	// 加载并验证配置文件
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建输出目录用于存储代理结果和工件
	if err := os.MkdirAll(cfg.Execution.OutputDir, 0755); err != nil {
		log.Printf("创建输出目录失败: %v", err)
	}

	// 如果配置了日志记录，则创建日志目录
	// TODO: 应使用filepath.Dir(cfg.Logging.File)而不是硬编码"logs"
	if logDir := "logs"; cfg.Logging.File != "" {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			log.Printf("创建日志目录失败: %v", err)
		}
	}

	// 将config.LLMConfig转换为models.LLMConfig以确保类型兼容
	// 这是必要的，因为不同包定义了各自的LLMConfig类型
	llmConfig := &models.LLMConfig{
		Provider:    cfg.LLM.Provider,    // LLM提供者（如"openai", "mock"）
		Model:       cfg.LLM.Model,       // 模型名称（如"gpt-3.5-turbo"）
		Temperature: cfg.LLM.Temperature, // 响应随机性（0.0-2.0）
		MaxTokens:   cfg.LLM.MaxTokens,   // 响应最大令牌数
		APIKey:      cfg.LLM.APIKey,      // API认证密钥
		BaseURL:     cfg.LLM.BaseURL,     // 自定义API端点URL
	}

	// DynAgent 现在内部管理LLM提供者，不需要外部创建
	// 所有智能组件都在 DynAgent.initializeComponents() 中初始化

	// 配置代理的所有必要参数
	agentConfig := &models.AgentConfig{
		Name:            cfg.Agent.Name,            // 代理标识符
		RoleDescription: cfg.Agent.RoleDescription, // 代理角色和能力描述
		MaxSteps:        cfg.Agent.MaxSteps,        // 最大规划深度层级
		MaxRounds:       cfg.Agent.MaxRounds,       // 最大对话轮数
		Parallel:        cfg.Agent.Parallel,        // 启用并行任务执行
		LLMConfig:       *llmConfig,                // LLM提供者配置
		Tools:           cfg.Tools.Enabled,         // 启用的工具列表
		OutputDir:       cfg.Execution.OutputDir,   // 输出文件目录
	}

	// 使用配置创建并初始化代理
	// 代理负责任务规划、执行和结果管理
	myAgent := agent.NewDynAgent(agentConfig)
	// DynAgent 在 NewDynAgent 中已经自动初始化组件
	log.Printf("代理 %s 初始化完成", myAgent.GetName())

	// 为所有操作创建后台上下文
	ctx := context.Background()

	// 根据命令行参数确定运行模式
	if *query != "" {
		// 非交互模式：处理单个查询后退出
		processQuery(ctx, myAgent, *query)
	} else if *interactive {
		// 交互模式：启动对话循环
		runInteractive(ctx, myAgent)
	} else {
		// 未指定有效模式，显示用法
		fmt.Println("使用 -q 进行单查询模式或 -i 进行交互模式")
		flag.Usage()
	}
}

// runInteractive 启动与代理的交互会话
// 用户可以实时输入查询、命令并接收响应
func runInteractive(ctx context.Context, dynAgent *agent.DynAgent) {
	// 显示欢迎横幅和可用命令
	fmt.Println("╭─────────────────────────────────────╮")
	fmt.Println("│        Go智能代理框架               │")
	fmt.Println("├─────────────────────────────────────┤")
	fmt.Println("│  命令:                              │")
	fmt.Println("│  • exit/quit - 退出程序             │")
	fmt.Println("│  • clear - 清屏                     │")
	fmt.Println("│  • help - 显示帮助信息              │")
	fmt.Println("╰─────────────────────────────────────╯")
	fmt.Println()

	// 创建缓冲读取器处理用户输入
	reader := bufio.NewReader(os.Stdin)

	// 主交互循环
	for {
		// 提示用户输入
		fmt.Print("用户> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("读取输入错误: %v\n", err)
			continue
		}

		// 清理输入并跳过空行
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// 在处理为查询前先处理特殊命令
		switch strings.ToLower(input) {
		case "exit", "quit":
			fmt.Println("\n再见！")
			return
		case "clear":
			clearScreen()
			continue
		case "help":
			showHelp()
			continue
		}

		// 将用户输入作为代理查询进行处理
		processQuery(ctx, dynAgent, input)
		fmt.Println() // 在交互间添加间距
	}
}

// processQuery 通过代理系统处理单个查询
// 显示查询内容、处理并展示结果
func processQuery(ctx context.Context, dynAgent *agent.DynAgent, query string) {
	// 显示正在处理的查询内容，带视觉分隔符
	fmt.Printf("\n正在处理: %s\n", query)
	fmt.Println(strings.Repeat("-", 50))

	// 将查询发送给代理进行处理
	result, err := dynAgent.ProcessMessage(ctx, query)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	// 显示代理响应
	fmt.Printf("\n代理> %s\n", result.Message)

	// 只有实际保存了文件才显示保存信息
	if len(result.AllFiles) > 0 {
		fmt.Printf("\n输出文件已保存至:")
		for _, filePath := range result.AllFiles {
			fmt.Printf(" %s", filePath)
		}
		fmt.Println()
	}
}

// clearScreen 使用ANSI转义码清除终端屏幕
// 适用于类Unix系统（Linux、macOS）
func clearScreen() {
	// ANSI转义序列：\033[H（光标回到首行）+ \033[2J（清屏）
	fmt.Print("\033[H\033[2J")
	fmt.Println("屏幕已清除。")
}

// showHelp 显示交互模式的详细帮助信息
// 展示可用命令和使用说明
func showHelp() {
	fmt.Println("\n═══════════════════════════════════════")
	fmt.Println("                 帮助                  ")
	fmt.Println("═══════════════════════════════════════")
	fmt.Println()
	fmt.Println("可用命令:")
	fmt.Println("  exit, quit - 退出程序")
	fmt.Println("  clear      - 清除屏幕")
	fmt.Println("  help       - 显示此帮助信息")
	fmt.Println()
	fmt.Println("直接输入您的问题或任务即可与代理交互。")
	fmt.Println()
}
