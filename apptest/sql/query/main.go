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

type sqlResponse struct {
	Error string        `json:"error"`
	Data  []sqlDataItem `json:"data"`
}

type sqlDataItem struct {
	Labels     map[string]string `json:"labels"`
	DataPoints []struct {
		Timestamp int64   `json:"timestamp"`
		Value     float64 `json:"value"`
	} `json:"datapoints"`
}

type Logger interface {
	LogRequest(r *http.Request, start time.Time)
}

type DefaultLogger struct{}

func (l *DefaultLogger) LogRequest(r *http.Request, start time.Time) {
	fmt.Printf("收到请求时间: %s\n", start.Format("2006-01-02 15:04:05.000"))
	fmt.Printf("收到请求路径: %s\n", r.URL.Path)
	fmt.Printf("请求方法: %s\n", r.Method)
	fmt.Println("查询参数:", r.URL.Query())
	fmt.Println("请求头:", r.Header)

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

type Server struct {
	logger Logger
	addr   string
}

func NewServer(addr string) *Server {
	return &Server{
		logger: &DefaultLogger{},
		addr:   addr,
	}
}

func (s *Server) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		s.logger.LogRequest(r, start)
		next.ServeHTTP(w, r)
		fmt.Printf("处理耗时: %v\n", time.Since(start))
	}
}

func generateSingleDataPoint() sqlResponse {
	now := time.Now().Unix()
	return sqlResponse{
		Error: "",
		Data: []sqlDataItem{
			{
				Labels: map[string]string{
					"instance": "localhost:5001",
					"job":      "sql",
				},
				DataPoints: []struct {
					Timestamp int64   `json:"timestamp"`
					Value     float64 `json:"value"`
				}{
					{Timestamp: now, Value: 42.5},
				},
			},
		},
	}
}

func generateMultipleDataPoints() sqlResponse {
	now := time.Now().Unix()
	return sqlResponse{
		Error: "",
		Data: []sqlDataItem{
			{
				Labels: map[string]string{
					"instance": "localhost:5001",
					"job":      "sql",
				},
				DataPoints: []struct {
					Timestamp int64   `json:"timestamp"`
					Value     float64 `json:"value"`
				}{
					{Timestamp: now - 60, Value: 42.5},
					{Timestamp: now - 30, Value: 43.2},
					{Timestamp: now, Value: 44.1},
				},
			},
		},
	}
}

func sendJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "编码响应失败", http.StatusInternalServerError)
	}
}

func main() {
	server := NewServer(":5001")

	http.HandleFunc("/", server.loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		sendJSONResponse(w, map[string]string{"message": "请求已接收"})
	}))

	http.HandleFunc("/sql/api/v1/query", server.loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		sendJSONResponse(w, generateSingleDataPoint())
	}))

	http.HandleFunc("/sql/api/v1/query_range", server.loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		sendJSONResponse(w, generateMultipleDataPoints())
	}))

	httpServer := &http.Server{
		Addr:         server.addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	fmt.Printf("服务器启动在 http://localhost%s\n", server.addr)
	log.Fatal(httpServer.ListenAndServe())
}
