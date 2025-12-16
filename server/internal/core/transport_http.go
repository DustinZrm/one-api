package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"one-mcp/internal/model"
)

// HTTPTransport implements Transport for wrapping a REST API as an MCP Tool
type HTTPTransport struct {
	Config     model.UpstreamServer
	ToolConfig ToolConfig
	
	onMessage func([]byte)
	onReady   func()
}

type ToolConfig struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Method      string          `json:"method"` // GET, POST
	Headers     map[string]string `json:"headers"`
	Parameters  []ToolParameter `json:"parameters"`
}

type ToolParameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // string, number, boolean
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
}

func NewHTTPTransport(cfg model.UpstreamServer) *HTTPTransport {
	var tc ToolConfig
	if cfg.ToolConfig != "" {
		json.Unmarshal([]byte(cfg.ToolConfig), &tc)
	}
	return &HTTPTransport{
		Config:     cfg,
		ToolConfig: tc,
	}
}

func (t *HTTPTransport) Start(ctx context.Context, onMessage func([]byte), onReady func()) error {
	t.onMessage = onMessage
	t.onReady = onReady

	// Immediately signal ready as we are a virtual server
	if t.onReady != nil {
		go t.onReady()
	}

	// Block until context cancelled
	<-ctx.Done()
	return nil
}

func (t *HTTPTransport) Send(payload []byte) error {
	var req JSONRPCMessage
	if err := json.Unmarshal(payload, &req); err != nil {
		return err
	}

	// Handle standard MCP requests
	if req.Method == "initialize" {
		t.handleInitialize(req.ID)
		return nil
	}
	if req.Method == "notifications/initialized" {
		return nil
	}
	if req.Method == "ping" {
		t.reply(req.ID, "pong")
		return nil
	}
	if req.Method == "tools/list" {
		t.handleToolsList(req.ID)
		return nil
	}
	if req.Method == "tools/call" {
		t.handleToolCall(req.ID, req.Params)
		return nil
	}

	// Unknown method
	// t.replyError(req.ID, -32601, "Method not found")
	return nil
}

func (t *HTTPTransport) Close() error {
	return nil
}

// Helpers

func (t *HTTPTransport) reply(id *json.RawMessage, result interface{}) {
	if id == nil {
		return
	}
	resBytes, _ := json.Marshal(result)
	rawRes := json.RawMessage(resBytes)
	
	resp := JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result:  rawRes,
	}
	payload, _ := json.Marshal(resp)
	if t.onMessage != nil {
		t.onMessage(payload)
	}
}

func (t *HTTPTransport) replyError(id *json.RawMessage, code int, msg string) {
	if id == nil {
		return
	}
	resp := JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: msg,
		},
	}
	payload, _ := json.Marshal(resp)
	if t.onMessage != nil {
		t.onMessage(payload)
	}
}

func (t *HTTPTransport) handleInitialize(id *json.RawMessage) {
	// Return standard capabilities
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":    "one-mcp-http-wrapper",
			"version": "1.0.0",
		},
	}
	t.reply(id, result)
}

func (t *HTTPTransport) handleToolsList(id *json.RawMessage) {
	// Construct JSON Schema from parameters
	properties := make(map[string]interface{})
	required := []string{}

	for _, p := range t.ToolConfig.Parameters {
		// Only expose parameters that:
		// 1. Don't have a default value OR
		// 2. Have a default value but we want to allow LLM to override (Assuming yes)
		// User requirement: "User can fill default value, or leave empty for model to fill"
		// This implies if default is provided, it's optional for the model?
		// Let's assume:
		// - If Required=true, it goes into required list (unless Default is set?)
		// - Actually, if Default is set in our config, the Model doesn't NEED to provide it.
		// - So if Default != "", we treat it as optional for the Model.
		
		prop := map[string]interface{}{
			"type":        p.Type,
			"description": p.Description,
		}
		if p.Default != "" {
			prop["default"] = p.Default
		}
		
		properties[p.Name] = prop
		
		if p.Required && p.Default == "" {
			required = append(required, p.Name)
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}

	tool := map[string]interface{}{
		"name":        t.ToolConfig.Name,
		"description": t.ToolConfig.Description,
		"inputSchema": schema,
	}

	t.reply(id, map[string]interface{}{
		"tools": []interface{}{tool},
	})
}

func (t *HTTPTransport) handleToolCall(id *json.RawMessage, paramsRaw json.RawMessage) {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal(paramsRaw, &params); err != nil {
		t.replyError(id, -32700, "Parse error")
		return
	}

	if params.Name != t.ToolConfig.Name {
		t.replyError(id, -32601, "Tool not found")
		return
	}

	// Merge arguments with defaults
	finalArgs := make(map[string]interface{})
	
	// 1. Fill defaults
	for _, p := range t.ToolConfig.Parameters {
		if p.Default != "" {
			finalArgs[p.Name] = p.Default
		}
	}
	
	// 2. Override with provided args
	for k, v := range params.Arguments {
		finalArgs[k] = v
	}

	// Execute HTTP Request
	response, err := t.executeHTTPRequest(finalArgs)
	if err != nil {
		t.reply(id, map[string]interface{}{
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": fmt.Sprintf("Error executing HTTP request: %v", err),
				},
			},
			"isError": true,
		})
		return
	}

	t.reply(id, map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": response,
			},
		},
	})
}

func (t *HTTPTransport) executeHTTPRequest(args map[string]interface{}) (string, error) {
	targetURL := t.Config.URL
	method := t.ToolConfig.Method
	if method == "" {
		method = "GET"
	}

	var req *http.Request
	var err error

	if method == "GET" {
		// Append params to Query String
		u, err := url.Parse(targetURL)
		if err != nil {
			return "", err
		}
		q := u.Query()
		for k, v := range args {
			q.Set(k, fmt.Sprintf("%v", v))
		}
		u.RawQuery = q.Encode()
		req, err = http.NewRequest("GET", u.String(), nil)
	} else {
		// Send as JSON Body
		jsonBytes, _ := json.Marshal(args)
		req, err = http.NewRequest(method, targetURL, bytes.NewReader(jsonBytes))
		req.Header.Set("Content-Type", "application/json")
	}

	if err != nil {
		return "", err
	}

	// Add configured headers
	for k, v := range t.ToolConfig.Headers {
		req.Header.Set(k, v)
	}
	
	// Add Auth Token if exists
	if t.Config.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+t.Config.AuthToken)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return fmt.Sprintf("HTTP Error %d: %s", resp.StatusCode, string(bodyBytes)), nil
	}

	return string(bodyBytes), nil
}
