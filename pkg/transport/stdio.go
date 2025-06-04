package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/BearHuddleston/mcp-server-example/pkg/mcp"
)

// Stdio implements the stdio transport for MCP
type Stdio struct{}

// NewStdio creates a new stdio transport
func NewStdio() *Stdio {
	return &Stdio{}
}

// Start begins listening on stdin for JSON-RPC messages
func (t *Stdio) Start(ctx context.Context, server mcp.Server) error {
	log.Println("Starting stdio transport...")

	scanner := bufio.NewScanner(os.Stdin)

	// Create channels for message processing
	lineChan := make(chan string)
	errChan := make(chan error)

	// Start reader goroutine
	go func() {
		defer close(lineChan)
		defer close(errChan)

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case lineChan <- scanner.Text():
			}
		}

		if err := scanner.Err(); err != nil {
			select {
			case <-ctx.Done():
				return
			case errChan <- err:
			}
		}
	}()

	// Message processing loop
	for {
		select {
		case <-ctx.Done():
			log.Println("Stdio transport shutting down")
			return nil
		case err := <-errChan:
			if err != nil {
				log.Printf("Error reading input: %v", err)
			}
			return err
		case line, ok := <-lineChan:
			if !ok {
				log.Println("Input closed, exiting")
				return nil
			}

			if line == "" {
				continue
			}

			if err := t.handleMessage(ctx, server, line); err != nil {
				log.Printf("Error handling message: %v", err)
			}
		}
	}
}

// Stop stops the stdio transport (no-op for stdio)
func (t *Stdio) Stop() error {
	return nil
}

func (t *Stdio) handleMessage(ctx context.Context, server mcp.Server, line string) error {
	var req mcp.Request
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		return t.sendParseError(line, err)
	}

	// Validate JSON-RPC
	if req.JSONRPC != mcp.JSONRPCVersion {
		log.Printf("Invalid JSON-RPC version: %s", req.JSONRPC)
		return nil
	}

	// Handle notifications (no response expected)
	if req.ID == nil {
		log.Printf("Received notification: %s", req.Method)
		return nil
	}

	// Add stdout sender to context
	reqCtx := context.WithValue(ctx, mcp.ResponseSenderKey, &StdoutSender{})
	reqCtx, cancel := context.WithTimeout(reqCtx, 30*time.Second)
	defer cancel()

	return server.HandleRequest(reqCtx, req)
}

func (t *Stdio) sendParseError(line string, err error) error {
	// Try to extract ID from malformed JSON
	var errorID any = -1
	var partialReq map[string]any
	if json.Unmarshal([]byte(line), &partialReq) == nil {
		if id, exists := partialReq["id"]; exists && id != nil {
			errorID = id
		}
	}

	errorResp := mcp.Response{
		JSONRPC: mcp.JSONRPCVersion,
		ID:      errorID,
		Error: &mcp.ErrorResponse{
			Code:    mcp.ErrorCodeParseError,
			Message: "Parse error",
			Data:    err.Error(),
		},
	}

	respBytes, marshErr := json.Marshal(errorResp)
	if marshErr != nil {
		return marshErr
	}

	fmt.Println(string(respBytes))
	return nil
}

// StdoutSender implements ResponseSender for stdio transport
type StdoutSender struct{}

func (s *StdoutSender) SendResponse(response mcp.Response) error {
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}
	fmt.Println(string(jsonBytes))
	return nil
}

func (s *StdoutSender) SendError(id any, code int, message string, data any) error {
	errorResp := &mcp.ErrorResponse{
		Code:    code,
		Message: message,
		Data:    data,
	}
	response := mcp.Response{
		JSONRPC: mcp.JSONRPCVersion,
		ID:      id,
		Error:   errorResp,
	}
	return s.SendResponse(response)
}
