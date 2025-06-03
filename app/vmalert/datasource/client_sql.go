package datasource

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type sqlResponse struct {
	Message string        `json:"message"`
	Data    []sqlDataItem `json:"data"`
}

type sqlDataItem struct {
	Labels map[string]string `json:"labels"`
	Points []struct {
		Timestamp int64   `json:"timestamp"`
		Value     float64 `json:"value"`
	} `json:"points"`
}

func (r sqlResponse) metrics() []Metric {
	var ms []Metric
	for _, res := range r.Data {
		if len(res.Points) < 1 {
			continue
		}
		var m Metric
		for _, dp := range res.Points {
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
	if c.appendTypePrefix {
		r.URL.Path += "/sql"
	}
	if !*disablePathAppend {
		r.URL.Path += "/api/v1/sql_query"
	}
	q := r.URL.Query()
	q.Set("time", timestamp.Format(time.RFC3339))
	r.URL.RawQuery = q.Encode()
	c.setReqParams(r, query)
}

func (c *Client) setSqlRangeReqParams(r *http.Request, query string, start, end time.Time) {
	if c.appendTypePrefix {
		r.URL.Path += "/sql"
	}
	if !*disablePathAppend {
		r.URL.Path += "/api/v1/sql_query_range"
	}
	q := r.URL.Query()
	q.Add("start", start.Format(time.RFC3339))
	q.Add("end", end.Format(time.RFC3339))
	r.URL.RawQuery = q.Encode()
	c.setReqParams(r, query)
}
