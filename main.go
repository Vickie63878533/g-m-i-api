// main.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// ========= 1. 常量和全局变量定义 =========
// 和原始脚本中的常量对应

var (
	// User-Agent 列表
	userAgents = []string{
		 // Windows (已更新)
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36", // Windows 11/10 - Chrome
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 Edg/128.0.0.0", // Windows 11/10 - Edge
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:129.0) Gecko/20100101 Firefox/129.0", // Windows 11/10 - Firefox
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36 OPR/112.0.0.0", // Windows 11/10 - Opera

  // macOS (已更新)
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36", // macOS Sonoma (Intel) - Chrome
  "Mozilla/5.0 (Macintosh; Apple M1 Mac OS X 14_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Safari/605.1.15", // macOS Sonoma (Apple Silicon) - Safari
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 14.5; rv:129.0) Gecko/20100101 Firefox/129.0", // macOS Sonoma (Intel) - Firefox
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 Edg/128.0.0.0", // macOS Sonoma (Intel) - Edge
  "Mozilla/5.0 (Macintosh; Apple M2 Pro Mac OS X 14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36", // macOS Sonoma (Apple Silicon M2) - Chrome

  // Android (保持最新)
  "Mozilla/5.0 (Linux; Android 14; Pixel 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Mobile Safari/537.36", // Android 14 (Pixel 8 Pro) - Chrome
  "Mozilla/5.0 (Linux; Android 14; SM-S928B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Mobile Safari/537.36", // Android 14 (Samsung Galaxy S24 Ultra) - Chrome
  "Mozilla/5.0 (Linux; Android 13; a_real_phone_lol) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Mobile Safari/537.36", // Android 13 - Generic Device - Chrome
  "Mozilla/5.0 (Linux; Android 14; sdk_gphone64_arm64; rv:129.0) Gecko/129.0 Firefox/129.0", // Android 14 (Emulator) - Firefox

  // iOS (保持最新)
  "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1", // iPhone - iOS 18 - Safari
  "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/128.0.6613.25 Mobile/15E148 Safari/604.1", // iPhone - iOS 18 - Chrome
  "Mozilla/5.0 (iPad; CPU OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1", // iPad - iOS 18 - Safari
  "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/128.0 Mobile/15E148", // iPhone - iOS 17.5.1 - Firefox

  // Linux (保持最新)
  "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36", // Linux (Ubuntu/Debian) - Chrome
  "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:129.0) Gecko/20100101 Firefox/129.0", // Linux (Ubuntu) - Firefox
  "Mozilla/5.0 (X11; Fedora; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36", // Linux (Fedora) - Chrome
	}

	// 固定的认证 Token
	authToken = "gmi-free-2-api"

	// 创建一个可复用的 HTTP 客户端，并设置60秒超时
	// 这等同于 JS 版本中的 AbortController + setTimeout
	apiClient = &http.Client{
		Timeout: 60 * time.Second,
	}

	// 初始化随机数生成器
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// ========= 2. 用于 JSON 序列化的结构体 =========
// Go是强类型语言, 我们需要定义结构体来处理 JSON 数据

// Chat 请求体结构
type ChatRequest struct {
	Model       string       `json:"model"`
	Messages    []any        `json:"messages"` // 使用 any (interface{}) 兼容各种消息格式
	Stream      bool         `json:"stream"`
	Temperature *float64     `json:"temperature,omitempty"`
	MaxTokens   *int         `json:"max_tokens,omitempty"`
	TopP        *float64     `json:"top_p,omitempty"`
}

// 转发到上游服务的 Payload 结构
type ForwardPayload struct {
	Model       string   `json:"model"`
	Messages    []any    `json:"messages"`
	Stream      bool     `json:"stream"`
	Temperature float64  `json:"temperature"`
	MaxTokens   int      `json:"max_tokens"`
	TopP        float64  `json:"top_p"`
}

// 非流式响应的转换结构 (用于模拟 OpenAI 格式)
type FinalChatResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int64     `json:"created"`
	Model   string    `json:"model"`
	Choices []Choice  `json:"choices"`
	Usage   UsageData `json:"usage"`
}
type Choice struct {
	Index        int            `json:"index"`
	Message      ResponseMessage `json:"message"`
	FinishReason string         `json:"finish_reason"`
}
type ResponseMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type UsageData struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
// 上游服务返回的 JSON 结构 (仅需要我们关心的字段)
type UpstreamResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Result string    `json:"result"` // 备用字段
	Usage  UsageData `json:"usage"`
}


// ========= 3. 辅助函数 =========

// writeJSON 用于向客户端发送 JSON 响应
func writeJSON(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("Error writing JSON response: %v", err)
	}
}

// writeError 用于发送标准化的错误信息
func writeError(w http.ResponseWriter, statusCode int, message, errType string) {
	errObj := map[string]interface{}{
		"error": map[string]string{
			"message": message,
			"type":    errType,
		},
	}
	writeJSON(w, statusCode, errObj)
}

// randUA 从列表中随机选择一个 User-Agent
func randUA() string {
	return userAgents[rng.Intn(len(userAgents))]
}

// ========= 4. 核心处理器 (Handlers) =========

// handleModels 处理 /v1/models 的请求
func handleModels(w http.ResponseWriter, r *http.Request) {
	// 1. 创建到上游 API 的请求
	req, err := http.NewRequestWithContext(r.Context(), "GET", "https://api.gmi-serving.com/v1/models", nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create upstream request", "internal_server_error")
		return
	}

	// 2. 设置请求头
	req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ImNhNGNkNGU1LTMyY2YtNDQ5OC1hNDZiLTFiYjFmMzI3NTUzMiIsInNjb3BlIjoiaWVfbW9kZWwiLCJjbGllbnRJZCI6IjAwMDAwMDAwLTAwMDAtMDAwMC0wMDAwLTAwMDAwMDAwMDAwMCJ9.TTdQWMVpyx55Zb0oWqWcny1aYAl7yc_ctNmIphkkBfw")
	req.Header.Set("User-Agent", randUA())

	// 3. 发送请求
	resp, err := apiClient.Do(req)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Sprintf("Upstream API error: %v", err), "api_error")
		return
	}
	defer resp.Body.Close()

	// 4. 将上游的响应直接转发给客户端
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// handleChatCompletions 处理 /v1/chat/completions 的请求
func handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	// 1. 解码客户端请求体
	var chatReq ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&chatReq); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body", "invalid_request_error")
		return
	}

	// 2. 构建转发到上游服务的 payload，并设置默认值
	payload := ForwardPayload{
		Messages:    chatReq.Messages,
		Stream:      chatReq.Stream,
		Model:       "Qwen3-Coder-480B-A35B-Instruct-FP8",
		Temperature: 0.5,
		MaxTokens:   4096,
		TopP:        0.95,
	}
	if chatReq.Model != "" {
		payload.Model = chatReq.Model
	}
	if chatReq.Temperature != nil {
		payload.Temperature = *chatReq.Temperature
	}
	if chatReq.MaxTokens != nil {
		payload.MaxTokens = *chatReq.MaxTokens
	}
	if chatReq.TopP != nil {
		payload.TopP = *chatReq.TopP
	}

	// 3. 将 payload 序列化为 JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create forward payload", "internal_server_error")
		return
	}

	// 4. 创建到上游 API 的请求
	req, err := http.NewRequestWithContext(r.Context(), "POST", "https://console.gmicloud.ai/chat", bytes.NewReader(payloadBytes))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create upstream request", "internal_server_error")
		return
	}

	// 5. 设置所有需要的请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", randUA())
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Origin", "https://console.gmicloud.ai")
	req.Header.Set("Referer", "https://console.gmicloud.ai/playground/llm/qwen3-coder-480b-a35b-instruct-fp8/1c44de32-1a64-4fd6-959b-273ffefa0a6b?tab=playground")
	req.Header.Set("Sec-Ch-Ua", `"Not)A;Brand";v="8", "Chromium";v="138", "Google Chrome";v="138"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	// 6. 发送请求
	resp, err := apiClient.Do(req)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Sprintf("Upstream API error: %v", err), "api_error")
		return
	}
	defer resp.Body.Close()

	// 7. 根据是否为流式请求进行不同处理
	if payload.Stream {
		// 流式响应: 直接将上游的响应体 pipe 到客户端
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(resp.StatusCode)
		// io.Copy 会自动处理缓冲区，高效地将数据从源复制到目的地
		io.Copy(w, resp.Body)
	} else {
		// 非流式响应: 读取、转换、再发送
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to read upstream response", "internal_server_error")
			return
		}
		
		if resp.StatusCode != http.StatusOK {
			// 如果上游返回了错误, 直接将错误信息透传给客户端
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(resp.StatusCode)
			w.Write(bodyBytes)
			return
		}

		var upstreamResp UpstreamResponse
		if err := json.Unmarshal(bodyBytes, &upstreamResp); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to parse upstream response", "internal_server_error")
			return
		}

		// 构建最终的 OpenAI 格式响应
		finalResp := FinalChatResponse{
			ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   payload.Model,
			Choices: []Choice{
				{
					Index: 0,
					Message: ResponseMessage{
						Role:    "assistant",
						Content: "", // 默认值
					},
					FinishReason: "stop", // 默认值
				},
			},
			Usage: upstreamResp.Usage, // 直接使用上游的 usage
		}
		
		if len(upstreamResp.Choices) > 0 {
			finalResp.Choices[0].Message.Content = upstreamResp.Choices[0].Message.Content
			if upstreamResp.Choices[0].FinishReason != "" {
				finalResp.Choices[0].FinishReason = upstreamResp.Choices[0].FinishReason
			}
		} else if upstreamResp.Result != "" {
			// 兼容 result 字段
			finalResp.Choices[0].Message.Content = upstreamResp.Result
		}


		writeJSON(w, http.StatusOK, finalResp)
	}
}

// ========= 5. 中间件 (Middleware) =========

// corsAuthMiddleware 用于处理 CORS、认证和基本路由检查
func corsAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置 CORS 头
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// 处理 OPTIONS 预检请求
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 检查路径是否被允许
		if r.URL.Path != "/v1/chat/completions" && r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}

		// 检查认证
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer"))

		if token != authToken {
			writeError(w, http.StatusUnauthorized, "Invalid token", "authentication_error")
			return
		}

		// 如果一切正常，调用下一个处理器
		next.ServeHTTP(w, r)
	})
}

// ========= 6. 主函数 (main) =========

func main() {
	// 创建一个新的路由 Mux
	mux := http.NewServeMux()

	// 注册处理器函数
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		handleModels(w, r)
	})

	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		handleChatCompletions(w, r)
	})

	// 使用中间件包装主路由
	handler := corsAuthMiddleware(mux)

	// 定义服务器地址
	port := "8080"
	addr := ":" + port

	// 启动服务器
	log.Printf("🚀 Server starting on http://localhost:%s", port)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("💀 Could not start server: %s\n", err)
	}
}
