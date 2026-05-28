package repo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"aiops/internal/config"
	"aiops/internal/model"
)

// VMClient proxies queries to VictoriaMetrics
type VMClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewVMClient(cfg config.VMConfig) *VMClient {
	return &VMClient{
		baseURL: strings.TrimRight(cfg.URL, "/"),
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// vmResponse represents the VictoriaMetrics API response
type vmResponse struct {
	Status string   `json:"status"`
	Data   vmData   `json:"data"`
	Error  string   `json:"error,omitempty"`
}

type vmData struct {
	ResultType string          `json:"resultType"`
	Result     json.RawMessage `json:"result"`
}

type vmVectorResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"` // [timestamp, "value"]
}

type vmMatrixResult struct {
	Metric map[string]string `json:"metric"`
	Values [][]interface{}   `json:"values"` // [[timestamp, "value"], ...]
}

// QueryRange executes a PromQL range query
func (c *VMClient) QueryRange(q model.MetricsQuery) ([]model.MetricsQueryResult, error) {
	params := url.Values{}
	params.Set("query", q.Query)

	if q.Start != "" {
		params.Set("start", q.Start)
	} else {
		params.Set("start", fmt.Sprintf("%d", time.Now().Add(-1*time.Hour).Unix()))
	}
	if q.End != "" {
		params.Set("end", q.End)
	} else {
		params.Set("end", fmt.Sprintf("%d", time.Now().Unix()))
	}
	if q.Step != "" {
		params.Set("step", q.Step)
	} else {
		params.Set("step", "60s")
	}

	apiURL := fmt.Sprintf("%s/api/v1/query_range?%s", c.baseURL, params.Encode())
	return c.doQuery(apiURL)
}

// QueryInstant executes a PromQL instant query
func (c *VMClient) QueryInstant(query string, ts time.Time) ([]model.MetricsQueryResult, error) {
	params := url.Values{}
	params.Set("query", query)
	if !ts.IsZero() {
		params.Set("time", fmt.Sprintf("%d", ts.Unix()))
	}

	apiURL := fmt.Sprintf("%s/api/v1/query?%s", c.baseURL, params.Encode())
	return c.doQuery(apiURL)
}

func (c *VMClient) doQuery(apiURL string) ([]model.MetricsQueryResult, error) {
	resp, err := c.httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("vm request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read vm response: %w", err)
	}

	var vmResp vmResponse
	if err := json.Unmarshal(body, &vmResp); err != nil {
		return nil, fmt.Errorf("parse vm response: %w", err)
	}

	if vmResp.Status != "success" {
		return nil, fmt.Errorf("vm error: %s", vmResp.Error)
	}

	return c.parseResult(vmResp.Data)
}

func (c *VMClient) parseResult(data vmData) ([]model.MetricsQueryResult, error) {
	var results []model.MetricsQueryResult

	switch data.ResultType {
	case "vector":
		var vectors []vmVectorResult
		if err := json.Unmarshal(data.Result, &vectors); err != nil {
			return nil, fmt.Errorf("parse vector: %w", err)
		}
		for _, v := range vectors {
			r := model.MetricsQueryResult{Metric: v.Metric}
			if len(v.Value) >= 2 {
				ts, _ := v.Value[0].(float64)
				val, _ := v.Value[1].(string)
				var fval float64
				fmt.Sscanf(val, "%f", &fval)
				r.Values = []model.DataPoint{{Timestamp: time.Unix(int64(ts), 0), Value: fval}}
			}
			results = append(results, r)
		}

	case "matrix":
		var matrices []vmMatrixResult
		if err := json.Unmarshal(data.Result, &matrices); err != nil {
			return nil, fmt.Errorf("parse matrix: %w", err)
		}
		for _, m := range matrices {
			r := model.MetricsQueryResult{Metric: m.Metric}
			for _, v := range m.Values {
				if len(v) >= 2 {
					ts, _ := v[0].(float64)
					val, _ := v[1].(string)
					var fval float64
					fmt.Sscanf(val, "%f", &fval)
					r.Values = append(r.Values, model.DataPoint{Timestamp: time.Unix(int64(ts), 0), Value: fval})
				}
			}
			results = append(results, r)
		}
	}

	return results, nil
}
