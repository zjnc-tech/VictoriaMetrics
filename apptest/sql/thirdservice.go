package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
)

// DataPoint 定义数据点结构
type DataPoint struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

// SQLResponseTarget 定义单个目标的响应结构
type SQLResponseTarget struct {
	Labels     map[string]string `json:"labels"`
	DataPoints []DataPoint       `json:"datapoints"`
}

// SQLResponse 定义完整的响应结构
type SQLResponse []SQLResponseTarget

// 记录器接口
type Logger interface {
	LogRequest(r *http.Request, start time.Time)
	LogPrometheusWrite(r *http.Request, ts []prompb.TimeSeries)
}

// 默认记录器实现
type DefaultLogger struct{}

func (l *DefaultLogger) LogRequest(r *http.Request, start time.Time) {
	fmt.Printf("收到请求时间: %s\n", start.Format("2006-01-02 15:04:05.000"))
	fmt.Printf("收到请求路径: %s\n", r.URL.Path)
	fmt.Printf("请求方法: %s\n", r.Method)
	fmt.Println("查询参数:", r.URL.Query())
	fmt.Println("请求头:", r.Header)
}

func (l *DefaultLogger) LogPrometheusWrite(r *http.Request, ts []prompb.TimeSeries) {
	fmt.Printf("\n=== 收到 Prometheus 写入请求 ===\n")
	fmt.Printf("时间: %s\n", time.Now().Format("2006-01-02 15:04:05.000"))
	fmt.Printf("Content-Type: %s\n", r.Header.Get("Content-Type"))
	fmt.Printf("Content-Encoding: %s\n", r.Header.Get("Content-Encoding"))
	fmt.Printf("X-Prometheus-Remote-Write-Version: %s\n", r.Header.Get("X-Prometheus-Remote-Write-Version"))

	fmt.Printf("时间序列数量: %d\n", len(ts))
	for i, series := range ts {
		fmt.Printf("--- 时间序列 #%d ---\n", i+1)
		fmt.Println("标签:")
		for _, label := range series.Labels {
			fmt.Printf("  %s = %s\n", label.Name, label.Value)
		}
		fmt.Println("样本:")
		for _, sample := range series.Samples {
			t := time.Unix(0, sample.Timestamp*int64(time.Millisecond))
			fmt.Printf("  时间: %s, 值: %.4f\n", t.Format("2006-01-02 15:04:05.000"), sample.Value)
		}
	}
	fmt.Println("=== 请求处理完成 ===")
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

// 生成单个数据点的示例数据
func generateSingleDataPoint() SQLResponse {
	now := time.Now().Unix()
	return SQLResponse{
		{
			Labels: map[string]string{
				"instance": "localhost:5001",
				"job":      "sql",
			},
			DataPoints: []DataPoint{
				{Timestamp: now, Value: 42.5},
			},
		},
	}
}

// 生成多个数据点的示例数据
func generateMultipleDataPoints() SQLResponse {
	now := time.Now().Unix()
	return SQLResponse{
		{
			Labels: map[string]string{
				"instance": "localhost:5001",
				"job":      "sql",
			},
			DataPoints: []DataPoint{
				{Timestamp: now - 60, Value: 42.5},
				{Timestamp: now - 30, Value: 43.2},
				{Timestamp: now, Value: 44.1},
			},
		},
	}
}

// 发送JSON响应
func sendJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "编码响应失败", http.StatusInternalServerError)
	}
}

// 处理Prometheus远程写入请求
func (s *Server) handlePrometheusWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持 POST 请求", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-Type") != "application/x-protobuf" {
		http.Error(w, "只支持 application/x-protobuf 格式", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("读取请求体失败: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	decoded, err := snappy.Decode(nil, body)
	if err != nil {
		http.Error(w, fmt.Sprintf("解压数据失败: %v", err), http.StatusBadRequest)
		return
	}

	var writeRequest prompb.WriteRequest
	if err := proto.Unmarshal(decoded, &writeRequest); err != nil {
		http.Error(w, fmt.Sprintf("解析 Protobuf 数据失败: %v", err), http.StatusBadRequest)
		return
	}

	s.logger.LogPrometheusWrite(r, writeRequest.Timeseries)

	sendJSONResponse(w, map[string]string{"message": "数据已接收并处理"})
}

func main() {
	server := NewServer(":5001")

	// 注册路由处理器
	http.HandleFunc("/sql/api/v1/query", server.loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		sendJSONResponse(w, generateSingleDataPoint())
	}))

	http.HandleFunc("/sql/api/v1/query_range", server.loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		sendJSONResponse(w, generateMultipleDataPoints())
	}))

	http.HandleFunc("/alert/api/v1/webhook", server.loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		sendJSONResponse(w, map[string]string{"message": "请求已接收"})
	}))

	http.HandleFunc("/vector/api/v1/write", server.handlePrometheusWrite)

	// 配置HTTP服务器
	httpServer := &http.Server{
		Addr:         server.addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	fmt.Printf("服务器启动在 http://localhost%s\n", server.addr)
	log.Fatal(httpServer.ListenAndServe())
}
