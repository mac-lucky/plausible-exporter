package plausible_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/mac-lucky/plausible-exporter/plausible"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *plausible.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	base, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("failed to parse test server URL: %v", err)
	}
	return &plausible.Client{HostAPIBase: base, SiteID: "example.com", Token: "test-token"}
}

func TestGetTimeseriesData(t *testing.T) {
	clt := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/stats/aggregate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("unexpected Authorization header: %s", got)
		}
		if got := r.URL.Query().Get("site_id"); got != "example.com" {
			t.Errorf("unexpected site_id: %s", got)
		}
		if got := r.URL.Query().Get("metrics"); got != "visitors,pageviews,bounce_rate,visit_duration" {
			t.Errorf("unexpected metrics: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"results": {
				"pageviews": {"value": 42},
				"bounce_rate": {"value": 55.5},
				"visit_duration": {"value": 123.25},
				"visitors": {"value": 17}
			}
		}`))
	})

	data, err := clt.GetTimeseriesData()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.Pageviews != 42 {
		t.Errorf("Pageviews = %d, want 42", data.Pageviews)
	}
	if data.Visitors != 17 {
		t.Errorf("Visitors = %d, want 17", data.Visitors)
	}
	if data.BounceRate != 55.5 {
		t.Errorf("BounceRate = %v, want 55.5", data.BounceRate)
	}
	if data.VisitDuration != 123.25 {
		t.Errorf("VisitDuration = %v, want 123.25", data.VisitDuration)
	}
}

func TestGetTimeseriesDataHTTPError(t *testing.T) {
	clt := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	if _, err := clt.GetTimeseriesData(); err == nil {
		t.Fatal("expected an error for a 500 response, got nil")
	}
}

func TestGetTimeseriesDataMalformedJSON(t *testing.T) {
	clt := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	})

	if _, err := clt.GetTimeseriesData(); err == nil {
		t.Fatal("expected a decode error, got nil")
	}
}

func TestGetTimeseriesDataConnectionError(t *testing.T) {
	// Spin up and immediately close a server so the URL is well-formed but
	// nothing is listening on it, forcing http.DefaultClient.Do to fail.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	base, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("failed to parse test server URL: %v", err)
	}
	srv.Close()

	clt := &plausible.Client{HostAPIBase: base, SiteID: "example.com", Token: "test-token"}
	if _, err := clt.GetTimeseriesData(); err == nil {
		t.Fatal("expected a request error, got nil")
	}
}

func TestGetBreakdown(t *testing.T) {
	clt := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/stats/breakdown" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("property") != "event:goal" {
			t.Errorf("unexpected property: %s", q.Get("property"))
		}
		if q.Get("period") != "week" {
			t.Errorf("unexpected period: %s", q.Get("period"))
		}
		if q.Get("limit") != "20" {
			t.Errorf("unexpected limit: %s", q.Get("limit"))
		}
		_, _ = w.Write([]byte(`{
			"results": [
				{"goal": "Signup", "visitors": 10, "events": 12},
				{"goal": "Purchase", "visitors": 3, "events": 3},
				{"visitors": 1, "events": 1}
			]
		}`))
	})

	items, err := clt.GetBreakdown("event:goal", "goal", "week", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The third row has no "goal" key and must be dropped.
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].Name != "Signup" || items[0].Visitors != 10 || items[0].Events != 12 {
		t.Errorf("unexpected first item: %+v", items[0])
	}
	if items[1].Name != "Purchase" || items[1].Visitors != 3 || items[1].Events != 3 {
		t.Errorf("unexpected second item: %+v", items[1])
	}
}

func TestGetBreakdownDefaultPeriodAndNoLimit(t *testing.T) {
	clt := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("period") != "day" {
			t.Errorf("expected default period 'day', got %q", q.Get("period"))
		}
		if q.Has("limit") {
			t.Errorf("expected no limit param when limit=0, got %q", q.Get("limit"))
		}
		_, _ = w.Write([]byte(`{"results": []}`))
	})

	items, err := clt.GetBreakdown("event:props:theme", "theme", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("got %d items, want 0", len(items))
	}
}

func TestGetBreakdownHTTPError(t *testing.T) {
	clt := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	if _, err := clt.GetBreakdown("event:goal", "goal", "day", 0); err == nil {
		t.Fatal("expected an error for a 401 response, got nil")
	}
}

func TestGetBreakdownMalformedJSON(t *testing.T) {
	clt := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results": [`))
	})

	if _, err := clt.GetBreakdown("event:goal", "goal", "day", 0); err == nil {
		t.Fatal("expected a decode error, got nil")
	}
}

func TestGetHealth(t *testing.T) {
	clt := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/health" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"postgres": "ok", "clickhouse": "ok", "sites_cache": "fail"}`))
	})

	status, err := clt.GetHealth()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]bool{"postgres": true, "clickhouse": true, "sites_cache": false}
	if len(status) != len(want) {
		t.Fatalf("got %d components, want %d", len(status), len(want))
	}
	for k, v := range want {
		if status[k] != v {
			t.Errorf("status[%q] = %v, want %v", k, status[k], v)
		}
	}
}

func TestGetHealthHTTPError(t *testing.T) {
	clt := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	if _, err := clt.GetHealth(); err == nil {
		t.Fatal("expected an error for a 503 response, got nil")
	}
}

func TestGetHealthMalformedJSON(t *testing.T) {
	clt := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"postgres": `))
	})

	if _, err := clt.GetHealth(); err == nil {
		t.Fatal("expected a decode error, got nil")
	}
}
