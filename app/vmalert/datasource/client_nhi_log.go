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
	AggregationTs   []int64           `json:"timestamp"`
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
		m.Timestamps = metric.AggregationTs
		metrics = append(metrics, m)
	}
	return Result{Data: metrics}, nil
}

func (c *Client) setNhiLogReqParams(r *http.Request, query string, timestamp time.Time) error {
	if !*disablePathAppend {
		r.URL.Path += "/backends/api/v1/logs/query/stats"
	}
	q := r.URL.Query()
	if c.applyIntervalAsTimeFilter && c.evaluationInterval > 0 {
		q.Set("StartTime", strconv.FormatInt(timestamp.Add(-c.evaluationInterval).Unix(), 10))
		q.Set("EndTime", strconv.FormatInt(timestamp.Unix(), 10))
	} else {
		q.Set("Time", strconv.FormatInt(timestamp.Unix(), 10))
	}
	r.URL.RawQuery = q.Encode()
	return c.setFormDataParams(r, query)
}

func (c *Client) setNhiLogRangeReqParams(r *http.Request, query string, start, end time.Time) error {
	if !*disablePathAppend {
		r.URL.Path += "/backends/api/v1/logs/query/stats/range"
	}
	q := r.URL.Query()
	q.Add("StartTime", strconv.FormatInt(start.Unix(), 10))
	q.Add("EndTime", strconv.FormatInt(end.Unix(), 10))
	r.URL.RawQuery = q.Encode()
	return c.setFormDataParams(r, query)
}
