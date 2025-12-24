package core

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"one-mcp/internal/model"
	"gorm.io/gorm"
)

type Gateway struct {
	db        *gorm.DB
	upstreams map[string]*UpstreamClient // map[Name]*Client
	mu        sync.RWMutex
}

func NewGateway(db *gorm.DB) *Gateway {
	g := &Gateway{
		db:        db,
		upstreams: make(map[string]*UpstreamClient),
	}
	return g
}

func (g *Gateway) ReloadUpstreams() {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	// Stop existing
	for _, client := range g.upstreams {
		client.Stop()
	}
	g.upstreams = make(map[string]*UpstreamClient)
	
	var servers []model.UpstreamServer
	if err := g.db.Where("enabled = ?", true).Find(&servers).Error; err != nil {
		log.Printf("Failed to load upstreams: %v", err)
		return
	}
	
	for _, server := range servers {
		client := NewUpstreamClient(server)
		client.Start()
		g.upstreams[server.Name] = client
	}
}

func (g *Gateway) HandleMessage(msg []byte, allowedServerIDs []string, allowedTools []string) (*JSONRPCMessage, error) {
	fmt.Printf("[Gateway] Received message: %s\n", string(msg))
	var req JSONRPCMessage
	if err := json.Unmarshal(msg, &req); err != nil {
		fmt.Printf("[Gateway] JSON parse error: %v\n", err)
		return nil, err
	}
	
	// Create allowed map for fast lookup
	allowedSrv := make(map[string]bool)
	for _, id := range allowedServerIDs {
		allowedSrv[id] = true
	}

	allowedToolMap := make(map[string]bool)
	for _, t := range allowedTools {
		allowedToolMap[t] = true
	}
	
	// Permission check function
	hasPermission := func(srvID string, toolName string) bool {
		// 1. Check Tool Permission first (if configured)
		if len(allowedToolMap) > 0 {
			if allowedToolMap["*"] {
				return true
			}
			return allowedToolMap[toolName]
		}

		// 2. Fallback to Server Permission
		if len(allowedSrv) == 0 {
			return true // No restrictions implies all allowed? Or need explicit "*"
		}
		// If allowedServerIDs is empty, it usually means allow all in our previous logic, 
		// but let's stick to: if list provided, must match. If empty list, maybe allow all? 
		// Actually, let's assume if empty list provided in Session, it means ALL allowed 
		// ONLY IF the original DB string was empty.
		// For safety: empty allowedSrv means allow all.
		return allowedSrv[srvID]
	}
	
	switch req.Method {
	case "initialize":
		return g.handleInitialize(&req)
	case "notifications/initialized":
		return nil, nil
	case "tools/list":
		return g.handleToolsList(&req, hasPermission)
	case "tools/call":
		// Some clients (like Claude Desktop) might use "callTool" instead of "tools/call"?
		// No, standard is "tools/call". 
		// However, let's verify if the request params are coming in correctly.
		// Sometimes params are nested differently.
		return g.handleToolCall(&req, hasPermission)
	case "callTool": // Legacy or alternative method name handling
		return g.handleToolCall(&req, hasPermission)
	case "ping":
		// Handle ping (return pong usually, or empty result)
		return &JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  json.RawMessage([]byte("{}")),
		}, nil
	case "logging/setLevel":
		// Accept logging level changes but ignore them for now
		return &JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  json.RawMessage([]byte("{}")),
		}, nil
	case "completion/complete":
		// TODO: Implement completion forwarding if needed. 
		// For now, return empty completions to avoid client errors.
		emptyResult := map[string]interface{}{
			"completion": map[string]interface{}{
				"values":  []string{},
				"total":   0,
				"hasMore": false,
			},
		}
		resBytes, _ := json.Marshal(emptyResult)
		return &JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  resBytes,
		}, nil
	default:
		// Unknown method
		errResp := &JSONRPCError{Code: -32601, Message: "Method not supported"}
		return &JSONRPCMessage{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   errResp,
		}, nil
	}
}

func (g *Gateway) handleInitialize(req *JSONRPCMessage) (*JSONRPCMessage, error) {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": true,
			},
			"prompts": map[string]interface{}{
				"listChanged": false,
			},
			"resources": map[string]interface{}{
				"listChanged": false,
				"subscribe":   false,
			},
			"logging": map[string]interface{}{},
		},
		"serverInfo": map[string]string{
			"name":    "one-mcp-gateway",
			"version": "1.1.1",
		},
	}
	resBytes, _ := json.Marshal(result)
	return &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  resBytes,
	}, nil
}

func (g *Gateway) handleToolsList(req *JSONRPCMessage, hasPermission func(string, string) bool) (*JSONRPCMessage, error) {
	g.mu.RLock()
	clients := make([]*UpstreamClient, 0, len(g.upstreams))
	for _, c := range g.upstreams {
		clients = append(clients, c)
	}
	g.mu.RUnlock()

	var allTools []map[string]interface{}
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, client := range clients {
		wg.Add(1)
		go func(c *UpstreamClient) {
			defer wg.Done()
			
			var cursor string
			for {
				var resp *JSONRPCMessage
				var err error
				
				if cursor == "" {
					// Try sending nil first (no params)
					resp, err = c.Call("tools/list", nil)
				} else {
					resp, err = c.Call("tools/list", map[string]string{"cursor": cursor})
				}
				
				if err != nil {
					return
				}
				
				if resp.Error != nil {
					// Fallback Strategy for strict servers
					// 1. Try {} (empty object)
					// 2. Try {"cursor": null} (explicit null cursor)
					
					if cursor == "" && resp.Error.Code == -32602 {
						fmt.Printf("[Gateway] Upstream %s refused nil params, retrying with {}\n", c.Config.Name)
						resp, err = c.Call("tools/list", map[string]interface{}{})
						if err == nil && resp.Error != nil && resp.Error.Code == -32602 {
							fmt.Printf("[Gateway] Upstream %s refused {}, retrying with {\"cursor\": null}\n", c.Config.Name)
							resp, err = c.Call("tools/list", map[string]interface{}{"cursor": nil})
						}
						
						if err != nil || resp.Error != nil {
							fmt.Printf("[Gateway] Upstream %s failed all param attempts: %v\n", c.Config.Name, resp.Error)
							return
						}
					} else {
						fmt.Printf("[Gateway] Upstream %s returned error for tools/list: %v\n", c.Config.Name, resp.Error)
						return
					}
				}
				
				var result struct {
					Tools      []map[string]interface{} `json:"tools"`
					NextCursor string                   `json:"nextCursor"`
				}
				if err := json.Unmarshal(resp.Result, &result); err != nil {
					return
				}

				// Prefix tool names
				for _, tool := range result.Tools {
					if name, ok := tool["name"].(string); ok {
						prefixedName := fmt.Sprintf("%s__%s", c.Config.Name, name)
						srvID := fmt.Sprintf("%d", c.Config.ID)
						
						// Check Permission
						if hasPermission(srvID, prefixedName) {
							tool["name"] = prefixedName
							mu.Lock()
							allTools = append(allTools, tool)
							mu.Unlock()
						}
					}
				}

				if result.NextCursor == "" {
					break
				}
				cursor = result.NextCursor
			}
		}(client)
	}
	wg.Wait()

	fmt.Printf("[Gateway] Aggregated %d tools\n", len(allTools))
	resBytes, _ := json.Marshal(map[string]interface{}{"tools": allTools})
	return &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  resBytes,
	}, nil
}

func (g *Gateway) handleToolCall(req *JSONRPCMessage, hasPermission func(string, string) bool) (*JSONRPCMessage, error) {
	fmt.Printf("[Gateway] Handling tool call: %s\n", string(req.Params))
	
	var params struct {
		Name string `json:"name"`
		Args interface{} `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		fmt.Printf("[Gateway] Failed to parse tool call params: %v\n", err)
		return nil, err
	}

	// Parse server name from tool name: serverName__toolName
	parts := strings.SplitN(params.Name, "__", 2)
	if len(parts) != 2 {
		return &JSONRPCMessage{
			JSONRPC: "2.0", ID: req.ID,
			Error: &JSONRPCError{Code: -32602, Message: "Invalid tool name format"},
		}, nil
	}
	
	serverName := parts[0]
	toolName := parts[1]

	g.mu.RLock()
	client, ok := g.upstreams[serverName]
	g.mu.RUnlock()

	if !ok {
		return &JSONRPCMessage{
			JSONRPC: "2.0", ID: req.ID,
			Error: &JSONRPCError{Code: -32602, Message: "Server not found"},
		}, nil
	}

	// Check permission
	srvID := fmt.Sprintf("%d", client.Config.ID)
	if !hasPermission(srvID, params.Name) {
		fmt.Printf("[Gateway] Permission denied for tool %s (Server ID: %s)\n", params.Name, srvID)
		return &JSONRPCMessage{
			JSONRPC: "2.0", ID: req.ID,
			Error: &JSONRPCError{Code: -32000, Message: "Permission denied"},
		}, nil
	}

	// Prepare upstream params
	upstreamParams := map[string]interface{}{
		"name":      toolName,
		"arguments": params.Args,
	}
	
	resp, err := client.Call("tools/call", upstreamParams)
	if err != nil {
		fmt.Printf("[Gateway] Upstream call failed: %v\n", err)
		return &JSONRPCMessage{
			JSONRPC: "2.0", ID: req.ID,
			Error: &JSONRPCError{Code: -32000, Message: err.Error()},
		}, nil
	}
	
	if resp.Error != nil {
		fmt.Printf("[Gateway] Upstream returned error: %v\n", resp.Error)
	}
	
	// Pass through result/error, but ensure ID matches request
	resp.ID = req.ID
	return resp, nil
}

func (g *Gateway) GetAllTools() ([]map[string]interface{}, error) {
	// Internal method to fetch all tools for admin UI
	// Bypass permission checks
	// Use handleToolsList with a permissive callback
	
	allowAll := func(srvID, toolName string) bool { return true }
	
	// Construct a fake request
	idRaw := json.RawMessage([]byte("0"))
	req := &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      &idRaw,
		Method:  "tools/list",
	}
	
	resp, err := g.handleToolsList(req, allowAll)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("rpc error: %s", resp.Error.Message)
	}
	
	var result struct {
		Tools []map[string]interface{} `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}
	
	return result.Tools, nil
}

