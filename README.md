# Go Agent Framework

A modular intelligent agent system in Go for complex task planning and execution.

## Features

- **Multi-level Task Planning**: Hierarchical task decomposition with configurable depth
- **Tool Integration**: Built-in tools for calculation, file operations, and search
- **LLM Provider Support**: OpenAI and mock providers with easy extensibility
- **Modular Architecture**: Clean separation of concerns with interfaces
- **Interactive CLI**: User-friendly command-line interface

## Quick Start

```bash
# 1. Set your OpenAI API key
export OPENAI_API_KEY="your-api-key"

# 2. Initialize the project
make init

# 3. Build the agent
make build

# 4. Run in interactive mode
make run
```

## Project Structure

```
go-agent/
├── cmd/main.go          # CLI entry point
├── pkg/                 # Core packages
│   ├── agent/          # Agent orchestration
│   ├── planning/       # Task planning
│   ├── execution/      # Task execution
│   ├── llm/           # LLM providers
│   ├── tools/         # Built-in tools
│   └── models/        # Data structures
├── examples/           # Usage examples
├── config.yaml        # Configuration
└── Makefile          # Build automation
```

## Usage

### Interactive Mode
```bash
make run
# or
./bin/go-agent -i
```

### Single Query
```bash
./bin/go-agent -q "What is the capital of France?"
```

### Run Example
```bash
make example
```

## Configuration

Edit `config.yaml` to customize:

```yaml
agent:
  name: "Go-Agent"
  max_steps: [5, 3, 2]  # Planning depth
  max_rounds: 10        # Max conversation rounds

llm:
  provider: "openai"    # or "mock" for testing
  model: "gpt-4o-mini"
  temperature: 0.7
```

## Development

```bash
# Run tests
make test

# Format code
make fmt

# Clean artifacts
make clean
```

## Architecture

The framework follows a modular design:

1. **Agent**: Orchestrates the entire workflow
2. **Planner**: Decomposes complex tasks into subtasks
3. **Executor**: Executes tasks and invokes tools
4. **Tools**: Pluggable tools for specific operations
5. **LLM Provider**: Abstract interface for language models

## License

MIT