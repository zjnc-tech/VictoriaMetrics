package datasource

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type sqlResponse []sqlResponseTarget

type sqlResponseTarget struct {
	Labels     map[string]string `json:"labels"`
	DataPoints []struct {
		Timestamp int64   `json:"timestamp"`
		Value     float64 `json:"value"`
	} `json:"datapoints"`
}

func (r sqlResponse) metrics() []Metric {
	var ms []Metric
	for _, res := range r {
		if len(res.DataPoints) < 1 {
			continue
		}
		var m Metric
		for _, dp := range res.DataPoints {
			m.Values = append(m.Values, dp.Value)
			m.Timestamps = append(m.Timestamps, dp.Timestamp)
		}
		for k, v := range res.Labels {
			m.AddLabel(k, v)
		}
		ms = append(ms, m)
	}
	return ms
}

func parseSqlResponse(req *http.Request, resp *http.Response) (Result, error) {
	r := &sqlResponse{}
	if err := json.NewDecoder(resp.Body).Decode(r); err != nil {
		return Result{}, fmt.Errorf("error parsing sql metrics for %s: %w", req.URL.Redacted(), err)
	}
	return Result{Data: r.metrics()}, nil
}

func (c *Client) setSqlReqParams(r *http.Request, query string, timestamp time.Time) {
	// 如果启用了类型前缀,添加/sql到路径
	if c.appendTypePrefix {
		r.URL.Path += "/sql"
	}
	// 如果未禁用路径追加,添加/api/v1/query到路径
	if !*disablePathAppend {
		r.URL.Path += "/api/v1/query"
	}
	// 获取URL查询参数
	q := r.URL.Query()
	// 设置查询时间戳
	q.Set("time", timestamp.Format(time.RFC3339))
	// 编码URL查询参数
	r.URL.RawQuery = q.Encode()
	// 设置请求参数
	c.setReqParams(r, query)
}

func (c *Client) setSqlRangeReqParams(r *http.Request, query string, start, end time.Time) {
	if c.appendTypePrefix {
		r.URL.Path += "/sql"
	}
	if !*disablePathAppend {
		r.URL.Path += "/api/v1/query_range"
	}
	q := r.URL.Query()
	q.Add("start", start.Format(time.RFC3339))
	q.Add("end", end.Format(time.RFC3339))
	r.URL.RawQuery = q.Encode()
	c.setReqParams(r, query)
}
