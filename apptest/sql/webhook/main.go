package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// 记录器接口
type Logger interface {
	LogRequest(r *http.Request, start time.Time)
}

// 默认记录器实现
type DefaultLogger struct{}

func (l *DefaultLogger) LogRequest(r *http.Request, start time.Time) {
	fmt.Printf("收到请求时间: %s\n", start.Format("2006-01-02 15:04:05.000"))
	fmt.Printf("收到请求路径: %s\n", r.URL.Path)
	fmt.Printf("请求方法: %s\n", r.Method)
	fmt.Println("查询参数:", r.URL.Query())
	fmt.Println("请求头:", r.Header)

	// 读取并打印请求体数据
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Println("读取请求体失败:", err)
		} else {
			fmt.Println("请求体:", string(bodyBytes))
			// 重新设置请求体,因为ReadAll会消耗掉body
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
	}
}

// Server 定义HTTP服务器结构
type Server struct {
	logger Logger
	addr   string
}

// NewServer 创建新的服务器实例
func NewServer(addr string) *Server {
	return &Server{
		logger: &DefaultLogger{},
		addr:   addr,
	}
}

// 中间件：记录请求信息
func (s *Server) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		s.logger.LogRequest(r, start)
		next.ServeHTTP(w, r)
		fmt.Printf("处理耗时: %v\n", time.Since(start))
	}
}

// 发送JSON响应
func sendJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "编码响应失败", http.StatusInternalServerError)
	}
}

func main() {
	server := NewServer(":5003")

	http.HandleFunc("/alert/api/v1/webhook", server.loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		sendJSONResponse(w, map[string]string{"message": "请求已接收"})
	}))

	// 配置HTTP服务器
	httpServer := &http.Server{
		Addr:         server.addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	fmt.Printf("服务器启动在 http://localhost%s\n", server.addr)
	log.Fatal(httpServer.ListenAndServe())
}
