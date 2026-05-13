package plausible

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	HostAPIBase *url.URL
	SiteID      string
	Token       string
}

func (clt *Client) GetTimeseriesData() (*TimeseriesData, error) {
	url := clt.HostAPIBase.JoinPath("/api/v1/stats/aggregate")
	q := url.Query()
	q.Add("site_id", clt.SiteID)
	// TODO: Do we want to be able to configure this?
	q.Add("period", "day")
	q.Add("date", time.Now().UTC().Format("2006-01-02"))
	q.Add("metrics", "visitors,pageviews,bounce_rate,visit_duration")
	url.RawQuery = q.Encode()
	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", clt.Token))
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("plausible: request error: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 400 {
		return nil, fmt.Errorf("plausible: unexpected HTTP status code %d received", response.StatusCode)
	}

	defer response.Body.Close()
	var tsData tsDTO
	err = json.NewDecoder(response.Body).Decode(&tsData)
	if err != nil {
		return nil, fmt.Errorf("plausible: failed to decode response data: %w", err)
	}
	return tsData.ToTimeseriesData(), nil
}

// GetBreakdown queries /api/v1/stats/breakdown for an arbitrary property and
// returns one BreakdownItem per row. The property's display column is keyed
// in the JSON by the part after the last colon (e.g. event:goal → "goal";
// event:props:theme → "theme"), so the caller passes that key as columnKey.
// limit=0 means use the API default (100, max 1000).
func (clt *Client) GetBreakdown(property, columnKey, period string, limit int) ([]BreakdownItem, error) {
	u := clt.HostAPIBase.JoinPath("/api/v1/stats/breakdown")
	q := u.Query()
	q.Add("site_id", clt.SiteID)
	if period == "" {
		period = "day"
	}
	q.Add("period", period)
	q.Add("property", property)
	q.Add("metrics", "visitors,events")
	if limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", limit))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", clt.Token))
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("plausible: request error: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 400 {
		return nil, fmt.Errorf("plausible: breakdown %s unexpected HTTP status %d", property, response.StatusCode)
	}

	var raw struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.NewDecoder(response.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("plausible: failed to decode breakdown response: %w", err)
	}

	out := make([]BreakdownItem, 0, len(raw.Results))
	for _, row := range raw.Results {
		item := BreakdownItem{}
		if v, ok := row[columnKey].(string); ok {
			item.Name = v
		}
		if v, ok := row["visitors"].(float64); ok {
			item.Visitors = uint(v)
		}
		if v, ok := row["events"].(float64); ok {
			item.Events = uint(v)
		}
		if item.Name == "" {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

// GetHealth queries plausible's `/api/health` endpoint and returns a map of component names to their health status (true for healthy, false for unhealthy).
// Known components are postgres, clickhouse, sites_cache
func (clt *Client) GetHealth() (map[string]bool, error) {
	url := clt.HostAPIBase.JoinPath("/api/health")
	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("plausible: request error: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 400 {
		return nil, fmt.Errorf("plausible: unexpected HTTP status code %d received", response.StatusCode)
	}

	defer response.Body.Close()
	var healthStatus map[string]string
	err = json.NewDecoder(response.Body).Decode(&healthStatus)
	if err != nil {
		return nil, fmt.Errorf("plausible: failed to decode response data: %w", err)
	}
	var result = make(map[string]bool)
	for key, value := range healthStatus {
		result[key] = value == "ok"
	}
	return result, nil
}
