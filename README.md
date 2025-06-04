# MCP Coffee Server

A [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server implementation in Go that provides coffee shop information through tools, resources, and prompts, following Go project layout best practices.

## Features

- **MCP 2025-03-26 Specification Compliant**
- **Multiple Transport Support**: `stdio` (default, compatible with MCP Inspector), `http` (with SSE)
- **Coffee Shop Domain**: Tools, resources, and prompts for coffee shop operations
- **Graceful Shutdown & Configurable Timeouts**
- **Production Ready**: Structured logging, error handling, validation

## Project Structure

```
simple-mcp-server-refactored/
├── cmd/mcpserver/           # Application entrypoint
├── pkg/                     # Public library code
│   ├── mcp/                 # Core MCP protocol types
│   ├── config/              # Configuration management
│   ├── transport/           # Transport implementations
│   └── handlers/            # Domain-specific handlers
├── internal/server/         # Server implementation
└── go.mod
```

## Usage

```bash
# Build the application
go build -o mcpserver ./cmd/mcpserver

# Run with stdio transport (default)
./mcpserver

# Run with HTTP transport
./mcpserver -transport http -port 8080
```

  - `stdio`: Standard input/output (default, compatible with MCP Inspector)
  - `http`: HTTP with Server-Sent Events (SSE) support
- **Coffee Shop Domain**: Tools, resources, and prompts for coffee shop operations
- **Graceful Shutdown**: Proper signal handling and resource cleanup
- **Configurable Timeouts**: Request, shutdown, and HTTP timeouts
- **Production Ready**: Structured logging, error handling, and validation

## Quick Start

### Docker

You can run the MCP server using Docker:

1. **Build the Docker image**:
   ```bash
   docker build -t mcp-server .
   ```

2. **Run the container**:
   ```bash
   # For HTTP transport (exposes port 8080)
   docker run -p 8080:8080 mcp-server --transport http --port 8080
   
   # For stdio transport (useful with MCP Inspector)
   docker run -it mcp-server --transport stdio
   ```

3. **Using environment variables**:
   ```bash
   docker run -p 8080:8080 -e TRANSPORT=http -e PORT=8080 mcp-server
   ```

### Prerequisites

- Go 1.24+ installed
- For testing: [MCP Inspector](https://github.com/modelcontextprotocol/inspector)

### Installation

```bash
git clone <repository-url>
cd simple-mcp-server
go build
```

### Basic Usage

```bash
# Start with stdio transport (default)
go run ./...

# Start with HTTP transport
go run ./... --transport http --port 8080

# Custom configuration
go run ./... --transport http --port 9000 --request-timeout 45s
```

## Configuration

### Command Line Flags

| Flag | Description | Default | Example |
|------|-------------|---------|---------|
| `--transport` | Transport type (`stdio` or `http`) | `stdio` | `--transport http` |
| `--port` | HTTP port (ignored for stdio) | `8080` | `--port 9000` |
| `--request-timeout` | Request timeout duration | `30s` | `--request-timeout 45s` |

### Environment Variables

The server uses Go's built-in flag parsing. Configuration is primarily through command-line flags.

## Transports

### Stdio Transport

Perfect for command-line tools and MCP Inspector integration:

```bash
go run ./... --transport stdio
```

**Use Cases:**
- MCP Inspector debugging
- CLI integrations
- Development and testing

### HTTP Transport

RESTful HTTP API with optional Server-Sent Events:

```bash
go run ./... --transport http --port 8080
```

**Endpoints:**
- `POST /mcp` - Send JSON-RPC requests
- `GET /mcp` - Open SSE stream
- `GET /health` - Health check

**Examples:**

```bash
# Regular JSON response
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"initialize","id":1}'

# SSE stream response
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":"test"}'
```

## Implementing a New Handler

To add a new handler to the MCP server, follow these steps using the `getWeather` handler as an example:

1. **Create a new handler file** in `pkg/handlers/` (e.g., `weather.go`):

```go
package handlers

import (
	"context"
	"encoding/json"

	"github.com/your-org/simple-mcp-server-refactored/pkg/mcp"
)

type WeatherHandler struct {
	// Add any dependencies here (e.g., API clients, config)
}

// WeatherRequest represents the expected request parameters
type WeatherRequest struct {
	Location string `json:"location"`
}

// WeatherResponse represents the response structure
type WeatherResponse struct {
	Location    string  `json:"location"`
	Temperature float64 `json:"temperature"`
	Condition   string  `json:"condition"`
	Humidity    int     `json:"humidity"`
	WindSpeed   float64 `json:"wind_speed"`
	Unit        string  `json:"unit"`
}

// Handle processes the weather request
func (h *WeatherHandler) Handle(ctx context.Context, request json.RawMessage) (interface{}, error) {
	var req WeatherRequest
	if err := json.Unmarshal(request, &req); err != nil {
		return nil, mcp.NewInvalidParamsError("invalid request parameters")
	}

	// TODO: Implement actual weather data retrieval
	// This is a mock implementation
	return WeatherResponse{
		Location:    req.Location,
		Temperature: 72.5,
		Condition:   "Sunny",
		Humidity:    45,
		WindSpeed:   8.2,
		Unit:        "fahrenheit",
	}, nil
}

// Register registers the handler with the MCP server
func (h *WeatherHandler) Register(router *mcp.Router) {
	router.RegisterHandler("getWeather", h.Handle)
}
```

2. **Register the handler** in `internal/server/server.go`:

```go
// In NewServer function
weatherHandler := &handlers.WeatherHandler{}
weatherHandler.Register(router)
```

3. **Add tests** in `pkg/handlers/weather_test.go`:

```go
package handlers_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/your-org/simple-mcp-server-refactored/pkg/handlers"
	"github.com/stretchr/testify/assert"
)

func TestWeatherHandler(t *testing.T) {
	h := &handlers.WeatherHandler{}
	
	t.Run("successful request", func(t *testing.T) {
		req := map[string]interface{}{
			"location": "New York, NY",
		}
		reqBytes, _ := json.Marshal(req)

		result, err := h.Handle(context.Background(), reqBytes)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		
		resp, ok := result.(handlers.WeatherResponse)
		assert.True(t, ok)
		assert.Equal(t, "New York, NY", resp.Location)
	})

	t.Run("invalid request", func(t *testing.T) {
		req := map[string]interface{}{
			"invalid": "data",
		}
		reqBytes, _ := json.Marshal(req)

		_, err := h.Handle(context.Background(), reqBytes)
		assert.Error(t, err)
	})
}
```

4. **Update documentation** in the README.md to document the new handler.

## MCP Capabilities

### Tools

Interactive functions that can be called by the LLM:

| Tool | Description | Parameters |
|------|-------------|------------|
| `getDrinkNames` | Get list of available drinks | None |
| `getDrinkInfo` | Get detailed drink information | `name`: string (required) |

**Example:**
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "id": "1",
  "params": {
    "name": "getDrinkInfo",
    "arguments": {"name": "Latte"}
  }
}
```

### Resources

Contextual data managed by the application:

| Resource | URI | Description |
|----------|-----|-------------|
| `menu` | `menu://app` | Complete coffee shop menu |

**Example:**
```json
{
  "jsonrpc": "2.0",
  "method": "resources/read",
  "id": "1",
  "params": {"uri": "menu://app"}
}
```

### Prompts

Template-driven interactions for the LLM:

| Prompt | Description | Parameters |
|--------|-------------|------------|
| `drinkRecommendation` | Get personalized drink recommendations | `budget`: number (optional)<br>`preference`: string (optional) |
| `drinkDescription` | Get detailed drink descriptions | `drink_name`: string (required) |

**Example:**
```json
{
  "jsonrpc": "2.0",
  "method": "prompts/get",
  "id": "1",
  "params": {
    "name": "drinkRecommendation",
    "arguments": {"budget": 6, "preference": "sweet"}
  }
}
```

## Testing with MCP Inspector

1. **Install MCP Inspector:**
   ```bash
   npm install -g @modelcontextprotocol/inspector
   ```

2. **Start the inspector:**
   ```bash
   npx @modelcontextprotocol/inspector
   ```

3. **Connect to the server:**
   - **Transport**: `stdio`
   - **Command**: `go`
   - **Args**: `run ./...`


## Manual Testing
```bash
# Test stdio transport
echo '{"jsonrpc":"2.0","method":"initialize","id":1}' | go run ./... --transport stdio

# Test HTTP transport
go run ./... --transport http &
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"initialize","id":1}'
```

## API Reference

### JSON-RPC Methods

| Method | Description | Parameters |
|--------|-------------|------------|
| `initialize` | Initialize MCP session | Client info (optional) |
| `tools/list` | List available tools | None |
| `tools/call` | Execute a tool | `name`, `arguments` |
| `resources/list` | List available resources | None |
| `resources/read` | Read resource content | `uri` |
| `prompts/list` | List available prompts | None |
| `prompts/get` | Get prompt template | `name`, `arguments` (optional) |
| `ping` | Health check | None |

### Error Codes

| Code | Meaning | Description |
|------|---------|-------------|
| `-32700` | Parse Error | Invalid JSON was received |
| `-32600` | Invalid Request | Invalid JSON-RPC request |
| `-32601` | Method Not Found | Method does not exist |
| `-32602` | Invalid Params | Invalid method parameters |
| `-32603` | Internal Error | Internal JSON-RPC error |

### Systemd Service

```ini
[Unit]
Description=MCP Coffee Server
After=network.target

[Service]
Type=simple
User=mcp
ExecStart=/usr/local/bin/mcp-server --transport http --port 8080
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

## Troubleshooting

### Common Issues

**Connection Refused (HTTP)**
```bash
# Check if server is running
curl http://localhost:8080/health

# Verify port is not in use
lsof -i :8080
```

**Stdio Transport Not Responding**
```bash
# Check JSON format
echo '{"jsonrpc":"2.0","method":"ping","id":1}' | go run ./...
```

**Request Timeout**
```bash
# Increase timeout
go run ./... --request-timeout 60s
```

**Parse Errors**
- Ensure JSON is valid and properly formatted
- Check that all required fields are present
- Verify JSON-RPC 2.0 compliance

## Resources

- [Model Context Protocol Specification](https://modelcontextprotocol.io/specification/2025-03-26)
- [MCP Inspector](https://github.com/modelcontextprotocol/inspector)
- [Go Documentation](https://golang.org/doc/)
- [Jack Herrington's YouTube video on DIY MCP Server](https://www.youtube.com/watch?v=nTMSyldeVSw)

## Support

For issues and questions:
- Create an issue in the repository
- Check the [MCP documentation](https://modelcontextprotocol.io/)
- Review the [troubleshooting section](#troubleshooting)

## License

MIT