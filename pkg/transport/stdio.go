package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/BearHuddleston/mcp-server-template/pkg/config"
	"github.com/BearHuddleston/mcp-server-template/pkg/mcp"
)

// Stdio implements the stdio transport for MCP
type Stdio struct {
	config     *config.Config
	input      io.Reader
	output     io.Writer
	newScanner func(io.Reader) *bufio.Scanner
}

// NewStdio creates a new stdio transport
func NewStdio(cfg *config.Config) *Stdio {
	if cfg == nil {
		cfg = config.New()
	}
	return &Stdio{
		config: cfg,
		input:  os.Stdin,
		output: os.Stdout,
		newScanner: func(r io.Reader) *bufio.Scanner {
			return bufio.NewScanner(r)
		},
	}
}

// Start begins listening on stdin for JSON-RPC messages
func (t *Stdio) Start(ctx context.Context, server mcp.Server) error {
	slog.Info("starting stdio transport")

	scanner := t.newScanner(t.input)

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
			slog.Info("stdio transport shutting down")
			return nil
		case err := <-errChan:
			if err != nil {
				slog.Error("error reading input", "error", err)
			}
			return err
		case line, ok := <-lineChan:
			if !ok {
				slog.Info("input closed; exiting")
				return nil
			}

			if line == "" {
				continue
			}

			if err := t.handleMessage(ctx, server, line); err != nil {
				slog.Error("error handling message", "error", err)
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
		slog.Warn("invalid JSON-RPC version", "version", req.JSONRPC)
		return nil
	}

	// Handle notifications (no response expected)
	if req.ID == nil {
		slog.Info("received notification", "method", req.Method)
		return nil
	}

	// Add stdout sender to context
	reqCtx := context.WithValue(ctx, mcp.ResponseSenderKey, &StdoutSender{writer: t.output})
	reqCtx, cancel := context.WithTimeout(reqCtx, t.config.RequestTimeout)
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
		return errors.Join(err, marshErr)
	}

	if _, writeErr := fmt.Fprintln(t.output, string(respBytes)); writeErr != nil {
		return errors.Join(err, writeErr)
	}
	return nil
}

// StdoutSender implements ResponseSender for stdio transport
type StdoutSender struct {
	writer io.Writer
}

func (s *StdoutSender) resolveWriter() io.Writer {
	if s.writer != nil {
		return s.writer
	}
	return os.Stdout
}

func (s *StdoutSender) SendResponse(response mcp.Response) error {
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}
	if _, err := fmt.Fprintln(s.resolveWriter(), string(jsonBytes)); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}
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
