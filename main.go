// main.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// ========= 1. 常量和全局变量定义 =========
// 和原始脚本中的常量对应

var (
	// User-Agent 列表
	userAgents = []string{
		// Windows (已更新)
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",               // Windows 11/10 - Chrome
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 Edg/128.0.0.0", // Windows 11/10 - Edge
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:129.0) Gecko/20100101 Firefox/129.0",                                              // Windows 11/10 - Firefox
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36 OPR/112.0.0.0", // Windows 11/10 - Opera

		// macOS (已更新)
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",               // macOS Sonoma (Intel) - Chrome
		"Mozilla/5.0 (Macintosh; Apple M1 Mac OS X 14_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Safari/605.1.15",            // macOS Sonoma (Apple Silicon) - Safari
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 14.5; rv:129.0) Gecko/20100101 Firefox/129.0",                                              // macOS Sonoma (Intel) - Firefox
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 Edg/128.0.0.0", // macOS Sonoma (Intel) - Edge
		"Mozilla/5.0 (Macintosh; Apple M2 Pro Mac OS X 14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",        // macOS Sonoma (Apple Silicon M2) - Chrome

		// Android (保持最新)
		"Mozilla/5.0 (Linux; Android 14; Pixel 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Mobile Safari/537.36",      // Android 14 (Pixel 8 Pro) - Chrome
		"Mozilla/5.0 (Linux; Android 14; SM-S928B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Mobile Safari/537.36",         // Android 14 (Samsung Galaxy S24 Ultra) - Chrome
		"Mozilla/5.0 (Linux; Android 13; a_real_phone_lol) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Mobile Safari/537.36", // Android 13 - Generic Device - Chrome
		"Mozilla/5.0 (Linux; Android 14; sdk_gphone64_arm64; rv:129.0) Gecko/129.0 Firefox/129.0",                                        // Android 14 (Emulator) - Firefox

		// iOS (保持最新)
		"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",        // iPhone - iOS 18 - Safari
		"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/128.0.6613.25 Mobile/15E148 Safari/604.1", // iPhone - iOS 18 - Chrome
		"Mozilla/5.0 (iPad; CPU OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",                 // iPad - iOS 18 - Safari
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_5_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/128.0 Mobile/15E148",                    // iPhone - iOS 17.5.1 - Firefox

		// Linux (保持最新)
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",         // Linux (Ubuntu/Debian) - Chrome
		"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:129.0) Gecko/20100101 Firefox/129.0",                                // Linux (Ubuntu) - Firefox
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
	targetURL, _ := url.Parse("https://console.gmicloud.ai/chat")
	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// 自定义 Director 函数来修改请求
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req) // 执行默认的 Director 逻辑 (如设置 X-Forwarded-For 等)

		// 设置目标请求的 URL scheme, host 和 path
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		req.URL.Path = targetURL.Path

		// 修改 Host 头部
		req.Host = targetURL.Host

		model := "openai/gpt-oss-120b"

		if req.Body != nil {
			bodyBytes, readErr := io.ReadAll(req.Body)
			if readErr != nil {
				log.Printf("Error reading request body: %v. Forwarding without model injection.", readErr)
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			} else {
				req.Body.Close()
				var data map[string]interface{}
				if unmarshalErr := json.Unmarshal(bodyBytes, &data); unmarshalErr == nil {
					if modelValue, ok := data["model"]; ok {
						if modelValueStr, ok := modelValue.(string); ok {
							model = modelValueStr
						}
					}
					modifiedBodyBytes, marshalErr := json.Marshal(data)
					if marshalErr == nil {
						req.Body = io.NopCloser(bytes.NewBuffer(modifiedBodyBytes))
						req.ContentLength = int64(len(modifiedBodyBytes))
						req.GetBody = func() (io.ReadCloser, error) {
							return io.NopCloser(bytes.NewBuffer(modifiedBodyBytes)), nil
						}
						log.Printf("Successfully get model '%s' into request body.", model)
					} else {
						log.Printf("Error marshalling modified body: %v. Forwarding original body.", marshalErr)
						req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
						req.ContentLength = int64(len(bodyBytes))
						req.GetBody = func() (io.ReadCloser, error) {
							return io.NopCloser(bytes.NewBuffer(bodyBytes)), nil
						}
					}
				} else {
					log.Printf("Error unmarshalling request body: %v. Forwarding original body.", unmarshalErr)
					req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
					req.ContentLength = int64(len(bodyBytes))
					req.GetBody = func() (io.ReadCloser, error) {
						return io.NopCloser(bytes.NewBuffer(bodyBytes)), nil
					}
				}
			}
		} else if req.Body != nil {
			log.Printf("Request body present but Content-Type is not application/json ('%s'). Model not injected.", req.Header.Get("Content-Type"))
		}

		refrere := fmt.Sprintf("https://console.gmicloud.ai/playground/llm/%s/%s?tab=playground", model, uuid.New().String())

		newHeader := http.Header{}
		// 5. 设置所有需要的请求头
		newHeader.Set("Content-Type", "application/json")
		newHeader.Set("User-Agent", randUA())
		newHeader.Set("Accept", "application/json, text/plain, */*")
		newHeader.Set("Accept-Language", "zh-TW,zh;q=0.9")
		newHeader.Set("Cache-Control", "no-cache")
		newHeader.Set("Pragma", "no-cache")
		newHeader.Set("Origin", "https://console.gmicloud.ai")
		newHeader.Set("Referer", refrere)
		newHeader.Set("Sec-Ch-Ua", `"Not)A;Brand";v="8", "Chromium";v="137", "Google Chrome";v="137"`)
		newHeader.Set("Sec-Ch-Ua-Mobile", "?0")
		newHeader.Set("Sec-Ch-Ua-Platform", `"Windows"`)
		newHeader.Set("Sec-Fetch-Dest", "empty")
		newHeader.Set("Sec-Fetch-Mode", "cors")
		newHeader.Set("Sec-Fetch-Site", "same-origin")

		req.Header = newHeader
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.Request == nil {
			log.Println("WARN: ModifyResponse: resp.Request is nil. Cannot check for API key context.")
			return nil
		}

		if resp.StatusCode != http.StatusOK {
			// 1. 读取响应体以获取错误信息
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("ERROR: Received status %d, but failed to read response body: %v", resp.StatusCode, err)
				resp.Body.Close() // 即使读取失败，也要尝试关闭
				return err        // 返回一个错误，因为响应修改过程失败了
			}
			// 2. 读取后必须关闭原始的 Body
			_ = resp.Body.Close()

			// 3. 将读取的内容重新包装成一个新的 ReadCloser 放回 Body 中
			//    这样，调用这个代理的客户端才能接收到原始的错误响应
			resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			// 4. 记录完整的错误日志
			log.Printf(
				"ERROR: Upstream returned non-200 status. Status: %d, Body: %s",
				resp.StatusCode,
				string(bodyBytes),
			)
		}

		return nil
	}

	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		http.Error(rw, "Error forwarding request.", http.StatusBadGateway)
	}

	proxy.ServeHTTP(w, r)
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

	port := 7860
	valueStr := os.Getenv("PORT")
	if value, err := strconv.Atoi(valueStr); err == nil {
		port = value
	}

	addr := "0.0.0.0:" + strconv.Itoa(port)

	// 启动服务器
	log.Printf("🚀 Server starting on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("💀 Could not start server: %s\n", err)
	}
}
