# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go implementation of the DynAgent framework, a modular intelligent agent system designed for complex task planning and execution. This is a port of the Python DynAgentV2 architecture with Go-specific optimizations for performance and concurrency.

## Key Commands

### Build and Run
- `make build` - Build the agent binary to `bin/go-agent`
- `make run` - Run the agent in interactive mode
- `make quick-start` - Initialize project, check environment, and build
- `go run cmd/main.go` - Run directly without building

### Testing
- `make test` - Run all tests
- `make test-coverage` - Generate test coverage report
- `go test ./pkg/planning -v` - Test specific package
- `go test -run TestTaskPlanner_NeedsPlan` - Run specific test

### Development
- `make fmt` - Format all Go code
- `make lint` - Run code linting (requires golangci-lint)
- `make dev` - Format code and run tests
- `make clean` - Clean build artifacts and output directories

### Examples
- `make example` - Run simple agent example
- `make example-tools` - Run example with tool integration
- `go run examples/simple_agent.go` - Run specific example

### Configuration
- `make config` - Display current configuration
- `make check-env` - Verify required environment variables

## Configuration Management

### Configuration Files
- `config.yaml` - Default configuration with mock LLM provider
- `config-deepseek-v3.yaml` - DeepSeek-V3 specific configuration

### Environment Variables
- `OPENAI_API_KEY` - Required for OpenAI provider
- `OPENAI_BASE_URL` - Optional custom API endpoint

### Key Settings (from config.yaml)
```yaml
agent:
  max_steps: [5, 3, 2]  # Multi-level planning depth
  max_rounds: 10         # Maximum conversation rounds
  parallel: false        # Parallel/serial execution mode
llm:
  provider: "mock"       # Options: openai, deepseek-v3, mock
execution:
  timeout: 30           # Task execution timeout (seconds)
```

## Architecture

### Core Components

**DynAgent** (`pkg/agent/agent.go`):
- Main orchestrator implementing the Agent interface
- Coordinates planning, execution, and result processing
- Manages agent lifecycle and state

**TaskPlanner** (`pkg/planning/planner.go`):
- Analyzes tasks to determine if planning is needed
- Decomposes complex tasks into subtasks
- Supports multi-level recursive planning based on `max_steps` configuration

**TaskExecutor** (`pkg/execution/executor.go`):
- Executes individual tasks and manages tool invocations
- Analyzes task dependencies for parallel/serial execution
- Implements timeout and error handling

**RecordManager** (`pkg/record/manager.go`):
- Tracks hierarchical execution records
- Saves records to JSONL format in `records/` directory
- Supports parent-child relationships for nested tasks

**MessageManager** (`pkg/messaging/manager.go`):
- Maintains conversation history
- Manages system, user, and assistant messages
- Provides context for LLM calls

**LLMProvider** (`pkg/llm/provider.go`):
- Abstract interface for LLM integration
- Factory pattern for provider instantiation (`provider_factory.go`)
- Implementations: OpenAI (`openai_provider.go`), Mock provider for testing

### Tool System

**Built-in Tools** (`pkg/tools/`):
- `calculator_tool.go` - Basic math operations
- `file_tool.go` - File reading and writing
- `search_tool.go` - Web search simulation

**Tool Interface**:
```go
type Tool interface {
    GetName() string
    GetDescription() string
    Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
}
```

### Data Flow
```
User Input → DynAgent.ProcessMessage()
    ↓
TaskPlanner.Plan() → Task decomposition
    ↓
TaskExecutor.Execute() → Tool invocation
    ↓
RecordManager.Save() → Execution tracking
    ↓
Result formatting → User output
```

## Project Structure

```
go-agent/
├── cmd/main.go              # Entry point with CLI
├── pkg/                     # Core packages
│   ├── agent/              # Agent orchestration
│   ├── planning/           # Task planning logic
│   ├── execution/          # Task execution engine
│   ├── record/             # Execution records
│   ├── messaging/          # Message management
│   ├── llm/                # LLM providers
│   ├── tools/              # Built-in tools
│   ├── interfaces/         # Core interfaces
│   └── models/             # Data structures
├── internal/               # Internal implementation (legacy)
├── examples/               # Usage examples
├── config*.yaml            # Configuration files
└── Makefile               # Build automation
```

## Testing Guidelines

### Test File Locations
- Unit tests alongside source files: `*_test.go`
- Test data in `test_records/` directories
- Coverage reports: `coverage.html` after running `make test-coverage`

### Running Tests
```bash
# All tests
go test ./...

# Specific package with verbose output
go test -v ./pkg/planning

# With race detection
go test -race ./...

# Specific test function
go test -run TestTaskPlanner ./pkg/planning
```

## Common Development Tasks

### Adding a New Tool
1. Implement the `Tool` interface in `pkg/tools/`
2. Register in executor initialization
3. Add to `tools.enabled` in config.yaml
4. Write tests for the tool

### Adding a New LLM Provider
1. Implement `LLMProvider` interface in `pkg/llm/`
2. Add case in `provider_factory.go`
3. Update configuration schema
4. Add provider-specific config file

### Debugging
- Set `logging.level: "debug"` in config.yaml
- Check logs in `logs/agent.log`
- Use mock provider for deterministic testing
- Records saved in `records/` for execution replay

## Key Differences from Python Version

| Feature | Python (DynAgentV2) | Go Implementation |
|---------|-------------------|-------------------|
| Concurrency | asyncio/multiprocessing | goroutines/channels |
| Configuration | setting.py + JSON | YAML files |
| Record Storage | MongoDB + JSONL | File-based JSONL |
| SSE Support | Flask integration | Not yet implemented |
| Tool System | Dynamic imports | Interface-based registration |
| Type Safety | Runtime | Compile-time |

## Pending Features

Based on Python version capabilities not yet implemented:
- SSE (Server-Sent Events) streaming
- MongoDB integration for records
- MCP (Model Context Protocol) support
- Web API endpoints
- Citation management and deduplication
- Additional LLM providers (Gemini, Claude)
- Excel output generation
- Visual debugging interface

# Comment Guidelines
- Use Chinese comments for all Go code in this project
- Comments should explain the purpose and functionality, not just translate the code
- Function comments should describe what the function does and its parameters
- Package-level comments should explain the package's role in the system
- Inline comments should clarify complex logic or business rules
- Follow Go documentation conventions but in Chinese language

Example format:
```go
// functionName 函数的作用和功能描述
// 参数说明如有需要
func functionName() {
    // 复杂逻辑的解释
}
```