package datasource

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type nhiLogResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Metrics []Metric `json:"metrics"`
	} `json:"data"`
}

func parseNhiLogResponse(req *http.Request, resp *http.Response) (Result, error) {
	r := &nhiLogResponse{}
	if err := json.NewDecoder(resp.Body).Decode(r); err != nil {
		return Result{}, fmt.Errorf("error parsing sql metrics for %s: %w", req.URL.Redacted(), err)
	}
	return Result{Data: r.Data.Metrics}, nil
}

func (c *Client) setNhiLogReqParams(r *http.Request, query string, timestamp time.Time) error {
	if !*disablePathAppend {
		r.URL.Path += "/backends/api/v1/log/stats_query"
	}
	q := r.URL.Query()
	if c.applyIntervalAsTimeFilter && c.evaluationInterval > 0 {
		q.Set("StartTime", timestamp.Add(-c.evaluationInterval).Format(time.DateTime))
		q.Set("EndTime", timestamp.Format(time.DateTime))
	} else {
		q.Set("Time", timestamp.Format(time.DateTime))
	}
	r.URL.RawQuery = q.Encode()
	return c.setFormDataParams(r, query)
}

func (c *Client) setNhiLogRangeReqParams(r *http.Request, query string, start, end time.Time) error {
	if !*disablePathAppend {
		r.URL.Path += "/backends/api/v1/log/stats_query_range"
	}
	q := r.URL.Query()
	q.Add("StartTime", start.Format(time.DateTime))
	q.Add("EndTime", end.Format(time.DateTime))
	r.URL.RawQuery = q.Encode()
	return c.setFormDataParams(r, query)
}
