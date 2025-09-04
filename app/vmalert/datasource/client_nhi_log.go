package datasource

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type nhiLogResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Metrics []logMetric `json:"metrics"`
	} `json:"data"`
}

type logMetric struct {
	AggregationTs   []string          `json:"timestamp"`
	AggregationInfo map[string]string `json:"labels"`
	Values          []string          `json:"values"`
}

func parseNhiLogResponse(req *http.Request, resp *http.Response) (Result, error) {
	r := &nhiLogResponse{}
	if err := json.NewDecoder(resp.Body).Decode(r); err != nil {
		return Result{}, fmt.Errorf("error parsing sql metrics for %s: %w", req.URL.Redacted(), err)
	}
	var metrics []Metric
	for _, metric := range r.Data.Metrics {
		var m Metric
		for k, v := range metric.AggregationInfo {
			m.AddLabel(k, v)
		}
		for _, value := range metric.Values {
			valueFloat, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return Result{}, fmt.Errorf("error parsing value %s: %w", value, err)
			}
			m.Values = append(m.Values, valueFloat)
		}
		for _, timestamp := range metric.AggregationTs {
			timestamp, err := time.Parse(time.DateTime, timestamp)
			if err != nil {
				return Result{}, fmt.Errorf("error parsing timestamp %s: %w", timestamp, err)
			}
			m.Timestamps = append(m.Timestamps, timestamp.UnixMilli())
		}
		metrics = append(metrics, m)
	}
	return Result{Data: metrics}, nil
}

func (c *Client) setNhiLogReqParams(r *http.Request, query string, timestamp time.Time) error {
	if !*disablePathAppend {
		r.URL.Path += "/backends/api/v1/log/stats_query"
	}
	q := r.URL.Query()
	if c.applyIntervalAsTimeFilter && c.evaluationInterval > 0 {
		q.Set("StartTime", timestamp.Add(-c.evaluationInterval).Local().Format(time.DateTime))
		q.Set("EndTime", timestamp.Local().Format(time.DateTime))
	} else {
		q.Set("Time", timestamp.Local().Format(time.DateTime))
	}
	r.URL.RawQuery = q.Encode()
	return c.setFormDataParams(r, query)
}

func (c *Client) setNhiLogRangeReqParams(r *http.Request, query string, start, end time.Time) error {
	if !*disablePathAppend {
		r.URL.Path += "/backends/api/v1/log/stats_query_range"
	}
	q := r.URL.Query()
	q.Add("StartTime", start.Local().Format(time.DateTime))
	q.Add("EndTime", end.Local().Format(time.DateTime))
	r.URL.RawQuery = q.Encode()
	return c.setFormDataParams(r, query)
}
