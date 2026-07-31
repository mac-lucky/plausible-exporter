package prometheus

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/mac-lucky/plausible-exporter/plausible"
)

// NewServer registers its gauges on prometheus.DefaultRegisterer, so only one
// MetricsServer can be constructed per test binary - a second call panics on
// duplicate registration. Every case below is a subtest against one shared
// instance instead of its own, and the subtests run in the order below on
// purpose: the staleness checks build on state left by the update above them.
func TestMetricsServer(t *testing.T) {
	srv := NewServer([]string{"site-a", "site-b"})

	t.Run("UpdateDataForSite sets the per-site gauges", func(t *testing.T) {
		srv.UpdateDataForSite("site-a", &plausible.TimeseriesData{
			Pageviews:     100,
			Visitors:      42,
			BounceRate:    55.5,
			VisitDuration: 123.25,
		})

		if got := testutil.ToFloat64(srv.pageviews.WithLabelValues("site-a")); got != 100 {
			t.Errorf("pageviews = %v, want 100", got)
		}
		if got := testutil.ToFloat64(srv.visitors.WithLabelValues("site-a")); got != 42 {
			t.Errorf("visitors = %v, want 42", got)
		}
		if got := testutil.ToFloat64(srv.bounceRate.WithLabelValues("site-a")); got != 55.5 {
			t.Errorf("bounceRate = %v, want 55.5", got)
		}
		if got := testutil.ToFloat64(srv.visitDuration.WithLabelValues("site-a")); got != 123.25 {
			t.Errorf("visitDuration = %v, want 123.25", got)
		}
	})

	t.Run("UpdateHealthStatusForSite maps booleans to 1/0", func(t *testing.T) {
		srv.UpdateHealthStatusForSite(map[string]bool{"postgres": true, "clickhouse": false})

		if got := testutil.ToFloat64(srv.healthStatus.WithLabelValues("postgres")); got != 1 {
			t.Errorf("health[postgres] = %v, want 1", got)
		}
		if got := testutil.ToFloat64(srv.healthStatus.WithLabelValues("clickhouse")); got != 0 {
			t.Errorf("health[clickhouse] = %v, want 0", got)
		}
	})

	t.Run("UpdateGoalsForSite replaces stale series", func(t *testing.T) {
		srv.UpdateGoalsForSite("site-a", []plausible.BreakdownItem{
			{Name: "signup", Visitors: 5, Events: 6},
			{Name: "purchase", Visitors: 2, Events: 2},
		})
		if got := testutil.ToFloat64(srv.goalVisitors.WithLabelValues("site-a", "signup")); got != 5 {
			t.Errorf("goalVisitors[signup] = %v, want 5", got)
		}
		if got := testutil.ToFloat64(srv.goalEvents.WithLabelValues("site-a", "purchase")); got != 2 {
			t.Errorf("goalEvents[purchase] = %v, want 2", got)
		}

		// "purchase" drops out of the next breakdown; its series must not linger.
		srv.UpdateGoalsForSite("site-a", []plausible.BreakdownItem{
			{Name: "signup", Visitors: 7, Events: 8},
		})
		if got := testutil.ToFloat64(srv.goalVisitors.WithLabelValues("site-a", "signup")); got != 7 {
			t.Errorf("goalVisitors[signup] after refresh = %v, want 7", got)
		}
		if got := testutil.ToFloat64(srv.goalEvents.WithLabelValues("site-a", "purchase")); got != 0 {
			t.Errorf("goalEvents[purchase] should have been cleared, got %v", got)
		}
	})

	t.Run("UpdatePropForSite replaces stale series scoped to site and key", func(t *testing.T) {
		srv.UpdatePropForSite("site-a", "theme", []plausible.BreakdownItem{
			{Name: "dark", Visitors: 9, Events: 11},
		})
		if got := testutil.ToFloat64(srv.propVisitors.WithLabelValues("site-a", "theme", "dark")); got != 9 {
			t.Errorf("propVisitors[dark] = %v, want 9", got)
		}

		// "dark" drops out of the next breakdown; its series must not linger,
		// and a different site/key pair must be untouched by the delete.
		srv.UpdatePropForSite("site-b", "theme", []plausible.BreakdownItem{
			{Name: "dark", Visitors: 1, Events: 1},
		})
		srv.UpdatePropForSite("site-a", "theme", []plausible.BreakdownItem{
			{Name: "light", Visitors: 4, Events: 4},
		})
		if got := testutil.ToFloat64(srv.propVisitors.WithLabelValues("site-a", "theme", "dark")); got != 0 {
			t.Errorf("propVisitors[site-a,dark] should have been cleared, got %v", got)
		}
		if got := testutil.ToFloat64(srv.propVisitors.WithLabelValues("site-a", "theme", "light")); got != 4 {
			t.Errorf("propVisitors[light] = %v, want 4", got)
		}
		if got := testutil.ToFloat64(srv.propVisitors.WithLabelValues("site-b", "theme", "dark")); got != 1 {
			t.Errorf("propVisitors[site-b,dark] = %v, want 1 (unaffected by the site-a update)", got)
		}
	})
}
