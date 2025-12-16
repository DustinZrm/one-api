package core

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"one-mcp/internal/model"
)

// JSONRPC types
type JSONRPCMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *JSONRPCError    `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type UpstreamClient struct {
	Config    model.UpstreamServer
	transport Transport
	
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.RWMutex
	ready     bool

	// Request coordination
	pendingReqs map[string]chan JSONRPCMessage
	reqMu       sync.Mutex
	idCounter   int64
}

func NewUpstreamClient(cfg model.UpstreamServer) *UpstreamClient {
	ctx, cancel := context.WithCancel(context.Background())
	
	var transport Transport
	switch cfg.TransportType {
	case "stdio":
		transport = NewStdioTransport(cfg)
	case "sse", "streaminghttp": // Treat streaminghttp as SSE
		transport = NewSSETransport(cfg)
	case "http":
		transport = NewHTTPTransport(cfg)
	default:
		// Default to SSE for backward compatibility
		transport = NewSSETransport(cfg)
	}

	return &UpstreamClient{
		Config:      cfg,
		transport:   transport,
		ctx:         ctx,
		cancel:      cancel,
		pendingReqs: make(map[string]chan JSONRPCMessage),
	}
}

func (c *UpstreamClient) Stop() {
	c.cancel()
	c.transport.Close()
}

func (c *UpstreamClient) Start() {
	go c.connectLoop()
}

func (c *UpstreamClient) IsReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ready
}

// Call performs a synchronous JSON-RPC call to the upstream
func (c *UpstreamClient) Call(method string, params interface{}) (*JSONRPCMessage, error) {
	if !c.IsReady() && method != "initialize" {
		return nil, fmt.Errorf("upstream not ready")
	}

	id := atomic.AddInt64(&c.idCounter, 1)
	idStr := fmt.Sprintf("%d", id)
	idRaw := json.RawMessage([]byte(idStr))

	var paramsRaw json.RawMessage
	if params != nil {
		paramsBytes, _ := json.Marshal(params)
		paramsRaw = paramsBytes
		fmt.Printf("[Upstream %s] Calling %s with params: %s\n", c.Config.Name, method, string(paramsBytes))
	} else {
		fmt.Printf("[Upstream %s] Calling %s without params\n", c.Config.Name, method)
	}
	
	req := JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      &idRaw,
		Method:  method,
		Params:  paramsRaw,
	}

	respChan := make(chan JSONRPCMessage, 1)
	c.reqMu.Lock()
	c.pendingReqs[idStr] = respChan
	c.reqMu.Unlock()

	defer func() {
		c.reqMu.Lock()
		delete(c.pendingReqs, idStr)
		c.reqMu.Unlock()
	}()

	payload, _ := json.Marshal(req)
	if err := c.transport.Send(payload); err != nil {
		fmt.Printf("[Upstream %s] Send error: %v\n", c.Config.Name, err)
		return nil, err
	}

	select {
	case resp := <-respChan:
		// Log brief response info
		fmt.Printf("[Upstream %s] Received response for %s (ID: %s)\n", c.Config.Name, method, idStr)
		if resp.Error != nil {
			fmt.Printf("[Upstream %s] Response Error: %v\n", c.Config.Name, resp.Error)
		}
		return &resp, nil
	case <-time.After(30 * time.Second):
		fmt.Printf("[Upstream %s] Timeout waiting for %s (ID: %s)\n", c.Config.Name, method, idStr)
		return nil, fmt.Errorf("timeout waiting for upstream response")
	}
}

func (c *UpstreamClient) connectLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			fmt.Printf("[Upstream %s] Transport starting...\n", c.Config.Name)
			err := c.transport.Start(c.ctx, c.handleMessage, c.onTransportReady)
			
			c.mu.Lock()
			c.ready = false
			c.mu.Unlock()
			
			if err != nil {
				if c.ctx.Err() == nil {
					fmt.Printf("[Upstream %s] Transport error: %v. Retrying in 5s...\n", c.Config.Name, err)
					time.Sleep(5 * time.Second)
				}
			} else {
				fmt.Printf("[Upstream %s] Transport stopped normally.\n", c.Config.Name)
				if c.ctx.Err() == nil {
					time.Sleep(1 * time.Second)
				}
			}
		}
	}
}

func (c *UpstreamClient) onTransportReady() {
	c.mu.Lock()
	c.ready = true
	c.mu.Unlock()
	
	fmt.Printf("[Upstream %s] Transport ready. Initializing...\n", c.Config.Name)
	c.initialize()
}

func (c *UpstreamClient) initialize() {
	// Send initialize request to upstream to identify ourselves
	initParams := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"roots": map[string]interface{}{
				"listChanged": true,
			},
			"sampling": map[string]interface{}{},
		},
		"clientInfo": map[string]interface{}{
			"name":    "one-mcp-gateway",
			"version": "1.0.0",
		},
	}
	
	resp, err := c.Call("initialize", initParams)
	if err != nil {
		fmt.Printf("[Upstream %s] Initialization failed: %v\n", c.Config.Name, err)
		return
	}
	
	if resp.Error != nil {
		fmt.Printf("[Upstream %s] Initialization error: %v\n", c.Config.Name, resp.Error)
		return
	}
	
	// Send initialized notification
	notifyReq := JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	payload, _ := json.Marshal(notifyReq)
	c.transport.Send(payload)
	
	fmt.Printf("[Upstream %s] Initialized successfully\n", c.Config.Name)
}

func (c *UpstreamClient) handleMessage(msg []byte) {
	fmt.Printf("[Upstream %s] Received: %s\n", c.Config.Name, string(msg))
	var resp JSONRPCMessage
	if err := json.Unmarshal(msg, &resp); err != nil {
		fmt.Printf("[Upstream %s] Error parsing JSON: %v\n", c.Config.Name, err)
		return
	}

	if resp.ID != nil {
		// Response to a request
		var idVal interface{}
		if err := json.Unmarshal(*resp.ID, &idVal); err != nil {
			return
		}
		
		idStr := fmt.Sprintf("%v", idVal)
		
		c.reqMu.Lock()
		ch, ok := c.pendingReqs[idStr]
		c.reqMu.Unlock()
		
		if ok {
			ch <- resp
		}
	} else {
		// Notification - TODO: forward to gateway if needed
	}
}
