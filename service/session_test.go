package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Ljjgz110110/Agent-Platform/plugin-helper/xlog"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestSession(t *testing.T) {
	xl := xlog.NewLogger("test")
	session := NewSession("id")
	defer session.Close()

	mcpFileSystem := mockMcpServiceFileSystem(t)
	if mcpFileSystem == nil {
		t.Fatalf("mockMcpServiceFileSystem failed")
	}
	if err := mcpFileSystem.Start(xl); err != nil {
		t.Fatalf("mockMcpServiceFileSystem.Start failed: %v", err)
	}
	defer func() {
		err := mcpFileSystem.Stop(xl)
		if err != nil {
			t.Errorf("mockMcpServiceFileSystem.Stop failed: %v", err)
		}
	}()
	err := session.SubscribeSSE(xl, mcpFileSystem.Name, mcpFileSystem.GetSSEUrl())
	if err != nil {
		t.Fatalf("subscribeSSE failed: %v", err)
	}

	req := mcp.ListToolsRequest{
		PaginatedRequest: mcp.PaginatedRequest{
			Request: mcp.Request{
				Method: string(mcp.MethodToolsList),
			},
		},
	}
	c := session.GetEventChan()
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	err = session.sendToMcp(xl, mcpFileSystem.Name, mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      mcp.NewRequestId(1),
		Request: req.Request,
	}, b)
	if err != nil {
		t.Fatalf("sendToMcp failed: %v", err)
	}
	result := <-c
	if result.Data == "" {
		t.Fatalf("result.Data is nil")
	}
	xl.Infof("Received result: %v", result)
}

// TestSessionAggregatedToolsList 测试聚合工具列表功能
func TestSessionAggregatedToolsList(t *testing.T) {
	xl := xlog.NewLogger("test-aggregated-tools")
	session := NewSession("aggregated-test-id")
	defer session.Close()

	// 创建并启动第一个MCP服务
	mcpFileSystem := mockMcpServiceFileSystem(t)
	if mcpFileSystem == nil {
		t.Fatalf("mockMcpServiceFileSystem failed")
	}
	if err := mcpFileSystem.Start(xl); err != nil {
		t.Fatalf("mockMcpServiceFileSystem.Start failed: %v", err)
	}
	defer func() {
		err := mcpFileSystem.Stop(xl)
		if err != nil {
			t.Errorf("mockMcpServiceFileSystem.Stop failed: %v", err)
		}
	}()

	// 订阅第一个MCP
	err := session.SubscribeSSE(xl, mcpFileSystem.Name, mcpFileSystem.GetSSEUrl())
	if err != nil {
		t.Fatalf("subscribeSSE failed for fileSystem: %v", err)
	}

	// 创建工具列表请求
	toolsListReq := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      mcp.NewRequestId(1),
		Request: mcp.Request{
			Method: string(mcp.MethodToolsList),
		},
	}

	// 获取事件通道
	eventChan := session.GetEventChan()

	// 等待一小段时间确保通道设置完成
	time.Sleep(100 * time.Millisecond)

	// 发送聚合工具列表请�?
	reqBytes, err := json.Marshal(toolsListReq)
	if err != nil {
		t.Fatalf("Failed to marshal tools list request: %v", err)
	}

	xl.Infof("Sending tools list request")
	err = session.SendMessage(xl, reqBytes)
	if err != nil {
		t.Fatalf("Failed to send tools list message: %v", err)
	}
	xl.Infof("Tools list request sent")

	// 等待响应 (最多等�?0�?
	select {
	case result := <-eventChan:
		if result.Data == "" {
			t.Fatalf("result.Data is empty")
		}

		xl.Infof("Received aggregated tools result: %v", result.Data)

		// 解析响应以验证工具名是否有前缀
		var response mcp.JSONRPCResponse
		err = json.Unmarshal([]byte(result.Data), &response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// 检查结�?
		if response.Result == nil {
			t.Fatalf("Response result is nil")
		}

		// 验证工具列表结构
		resultBytes, err := json.Marshal(response.Result)
		if err != nil {
			t.Fatalf("Failed to marshal result: %v", err)
		}

		var toolsResult mcp.ListToolsResult
		err = json.Unmarshal(resultBytes, &toolsResult)
		if err != nil {
			t.Fatalf("Failed to unmarshal tools result: %v", err)
		}

		xl.Infof("Found %d tools in aggregated result", len(toolsResult.Tools))

		// 验证每个工具名都有MCP前缀
		foundPrefixed := false
		for _, tool := range toolsResult.Tools {
			xl.Infof("Tool: %s - %s", tool.Name, tool.Description)
			if strings.Contains(tool.Name, "_") {
				foundPrefixed = true
			}
		}

		if !foundPrefixed && len(toolsResult.Tools) > 0 {
			t.Errorf("No tools found with MCP prefix")
		}

	case <-time.After(10 * time.Second):
		t.Fatalf("Timeout waiting for aggregated tools list response")
	}

	// 验证工具列表已准备就�?
	if !session.IsToolsListReady() {
		t.Errorf("Tools list should be ready after receiving response")
	}

	// 验证可以获取聚合工具列表
	allTools := session.GetAllTools()
	if len(allTools) == 0 {
		t.Errorf("Expected some tools in aggregated list")
	}

	xl.Infof("Test completed successfully with %d aggregated tools", len(allTools))
}

func TestSessionInitializedNotificationNoResponse(t *testing.T) {
	xl := xlog.NewLogger("test-initialized-notification")
	session := NewSession("initialized-notification-test-id")
	t.Cleanup(func() {
		session.mu.Lock()
		delete(session.mcpClients, "test-mcp")
		session.mu.Unlock()
		session.Close()
	})

	session.mu.Lock()
	session.mcpClients["test-mcp"] = nil
	session.mu.Unlock()

	eventChan := session.GetEventChan()

	err := session.sendToMcp(
		xl,
		"test-mcp",
		mcp.JSONRPCRequest{
			JSONRPC: "2.0",
			Request: mcp.Request{Method: "notifications/initialized"},
		},
		[]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`),
	)
	if err != nil {
		t.Fatalf("failed to send initialized notification: %v", err)
	}

	select {
	case result := <-eventChan:
		t.Fatalf("expected no response event for initialized notification, got: %s", result.Data)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestSessionNoImmediateCleanupAfterLastChannelClosed(t *testing.T) {
	session := NewSession("no-immediate-cleanup")
	defer session.Close()

	cleanupCalled := make(chan string, 1)
	session.SetCleanupCallback(func(sessionId string) {
		cleanupCalled <- sessionId
	})

	_, closeChan := session.GetEventChanWithCloser()
	closeChan()

	select {
	case <-cleanupCalled:
		t.Fatalf("expected cleanup not to run immediately after last channel closed")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestSessionCleanupAfterNoConnectionTTL(t *testing.T) {
	session := NewSession("cleanup-after-ttl")
	defer session.Close()

	cleanupCalled := make(chan string, 1)
	session.SetCleanupCallback(func(sessionId string) {
		cleanupCalled <- sessionId
	})

	session.mu.Lock()
	session.LastReceiveTime = time.Now().Add(-sessionNoConnectionTTL - time.Second)
	session.mu.Unlock()

	session.checkInactivity()

	select {
	case got := <-cleanupCalled:
		if got != session.Id {
			t.Fatalf("expected session id %s, got %s", session.Id, got)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected cleanup callback to be triggered")
	}
}

func TestSessionNotCleanupBeforeNoConnectionTTL(t *testing.T) {
	session := NewSession("not-cleanup-before-ttl")
	defer session.Close()

	cleanupCalled := make(chan string, 1)
	session.SetCleanupCallback(func(sessionId string) {
		cleanupCalled <- sessionId
	})

	session.mu.Lock()
	session.LastReceiveTime = time.Now().Add(-sessionNoConnectionTTL + 10*time.Second)
	session.mu.Unlock()

	session.checkInactivity()

	select {
	case <-cleanupCalled:
		t.Fatalf("expected cleanup callback not to be triggered before TTL")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestSessionCustomCleanupConfig(t *testing.T) {
	session := newSession("custom-cleanup-config", sessionCleanupConfig{
		noConnectionTTL:         3 * time.Second,
		inactivityCheckInterval: time.Hour,
	})
	defer session.Close()

	cleanupCalled := make(chan string, 1)
	session.SetCleanupCallback(func(sessionId string) {
		cleanupCalled <- sessionId
	})

	session.mu.Lock()
	session.LastReceiveTime = time.Now().Add(-2 * time.Second)
	session.mu.Unlock()

	session.checkInactivity()

	select {
	case <-cleanupCalled:
		t.Fatalf("expected cleanup callback not to be triggered before custom TTL")
	case <-time.After(100 * time.Millisecond):
	}

	session.mu.Lock()
	session.LastReceiveTime = time.Now().Add(-4 * time.Second)
	session.mu.Unlock()

	session.checkInactivity()

	select {
	case <-cleanupCalled:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected cleanup callback to be triggered after custom TTL")
	}
}
