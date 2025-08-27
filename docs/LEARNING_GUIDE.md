# Go Agent 学习指南

## 🎯 学习目标

通过这个项目，您将学习到：

1. **Go语言基础语法**
   - 包管理和模块系统
   - 结构体和接口
   - 方法和函数
   - 错误处理
   - 并发编程（Goroutine和Channel）

2. **软件架构设计**
   - 模块化设计
   - 依赖注入
   - 接口抽象
   - 状态管理
   - 配置管理

3. **AI代理系统**
   - 任务规划和分解
   - LLM集成
   - 工具调用机制
   - 结果处理流程

## 📚 学习路径

### 第一阶段：Go语言基础 (1-2周)

#### 1. 基础语法
```bash
# 查看数据模型定义
cat internal/models/models.go
```

**学习要点：**
- `struct` 结构体定义
- `json` 标签的使用
- 指针和值传递
- 构造函数模式

#### 2. 包和模块
```bash
# 查看包结构
tree internal/
```

**学习要点：**
- 包的命名规范
- 导入和导出规则
- `go.mod` 文件管理
- 依赖管理

#### 3. 接口和方法
```bash
# 查看接口定义
cat internal/llm/call_llm.go
```

**学习要点：**
- 接口定义和实现
- 方法接收器
- 多态性应用

### 第二阶段：核心组件理解 (2-3周)

#### 1. 配置管理
```bash
# 运行配置示例
go run cmd/main.go config
```

**文件：** `internal/config/config.go`

**学习要点：**
- 结构体嵌套
- 默认值设置
- JSON序列化/反序列化
- 配置验证

#### 2. 消息管理
**文件：** `internal/messaging/message_manager.go`

**学习要点：**
- 消息构建模式
- 字符串处理
- 切片操作
- 格式化输出

#### 3. 状态管理
**文件：** `internal/state/state_manager.go`

**学习要点：**
- 文件I/O操作
- 状态持久化
- 并发安全（mutex）
- 错误处理

### 第三阶段：高级特性 (2-3周)

#### 1. 任务规划
**文件：** `internal/planning/task_planner.go`

**学习要点：**
- 复杂业务逻辑
- JSON解析
- 算法实现（循环依赖检查）
- 状态机模式

#### 2. 任务执行
**文件：** `internal/execution/task_executor.go`

**学习要点：**
- 并发执行（Goroutine）
- 同步机制（WaitGroup, Mutex）
- 回调函数
- 错误聚合

#### 3. 主控制器
**文件：** `internal/agent/dynagent.go`

**学习要点：**
- 组件协调
- 生命周期管理
- 复杂状态机
- 错误恢复

## 🛠️ 实践练习

### 练习1：简单使用
```bash
# 设置API密钥
export OPENAI_API_KEY="your-api-key"

# 运行简单示例
go run examples/simple_agent.go

# 运行工具示例
go run examples/with_tools.go

# 交互模式
go run cmd/main.go run
```

### 练习2：添加自定义工具
创建 `pkg/tools/my_tool.go`：

```go
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

### 练习3：扩展功能
1. 添加新的消息类型
2. 实现自定义LLM提供商
3. 添加数据库持久化
4. 实现Web API接口

## 🔧 调试技巧

### 1. 查看日志
```bash
# 启用详细日志
export GO_AGENT_DEBUG=true
go run cmd/main.go run
```

### 2. 状态检查
```bash
# 查看保存的状态
ls -la ./data/save_agent/
cat ./data/save_agent/[agent-id]/config.json
```

### 3. 性能分析
```go
import _ "net/http/pprof"

// 在main函数中添加
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

## 🚀 进阶项目

完成基础学习后，可以尝试：

1. **Web界面**：使用Gin框架创建Web API
2. **数据库集成**：添加PostgreSQL或MongoDB支持
3. **微服务架构**：拆分为多个服务
4. **容器化部署**：使用Docker部署
5. **监控系统**：添加Prometheus监控
6. **测试覆盖**：编写单元测试和集成测试

## 📖 推荐资源

### Go语言学习
- [Go官方教程](https://tour.golang.org/)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go by Example](https://gobyexample.com/)

### 架构设计
- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture/)

### AI代理系统
- [LangChain文档](https://docs.langchain.com/)
- [OpenAI API文档](https://platform.openai.com/docs/)

## 🤝 贡献指南

欢迎提交Pull Request！请确保：

1. 遵循Go代码规范
2. 添加适当的测试
3. 更新相关文档
4. 保持向后兼容性

## 📞 获取帮助

如果遇到问题：

1. 查看示例代码
2. 阅读错误信息
3. 检查配置设置
4. 查看日志输出

祝您学习愉快！🎉
