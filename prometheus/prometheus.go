package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/mac-lucky/plausible-exporter/plausible"
)

type MetricsServer struct {
	pageviews     *prometheus.GaugeVec
	visitors      *prometheus.GaugeVec
	bounceRate    *prometheus.GaugeVec
	visitDuration *prometheus.GaugeVec
	healthStatus  *prometheus.GaugeVec
	goalVisitors  *prometheus.GaugeVec
	goalEvents    *prometheus.GaugeVec
	propVisitors  *prometheus.GaugeVec
	propEvents    *prometheus.GaugeVec
}

func NewServer(siteIDs []string) *MetricsServer {
	pageviews := promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "plausible",
		Name:      "pageviews",
		Help:      "Number of page views for a given site",
	}, []string{"site_id"})

	visitors := promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "plausible",
		Name:      "visitors",
		Help:      "Number of visitors for a given site",
	}, []string{"site_id"})

	bounceRate := promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "plausible",
		Name:      "bounce_rate",
		Help:      "Bounce rate for a given site in %",
	}, []string{"site_id"})

	visitDuration := promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "plausible",
		Name:      "visit_duration",
		Help:      "Average visit duration for a given site in seconds",
	}, []string{"site_id"})

	healthStatus := promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "plausible",
		Name:      "health",
		Help:      "Health of the Plausible API (1 for healthy, 0 for unhealthy)",
	}, []string{"component"})

	goalVisitors := promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "plausible",
		Name:      "goal_visitors",
		Help:      "Unique visitors who triggered a conversion goal",
	}, []string{"site_id", "goal"})

	goalEvents := promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "plausible",
		Name:      "goal_events",
		Help:      "Total conversion events for a goal",
	}, []string{"site_id", "goal"})

	propVisitors := promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "plausible",
		Name:      "prop_visitors",
		Help:      "Unique visitors broken down by custom property value",
	}, []string{"site_id", "key", "value"})

	propEvents := promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "plausible",
		Name:      "prop_events",
		Help:      "Events broken down by custom property value",
	}, []string{"site_id", "key", "value"})

	return &MetricsServer{
		pageviews:     pageviews,
		visitors:      visitors,
		bounceRate:    bounceRate,
		visitDuration: visitDuration,
		healthStatus:  healthStatus,
		goalVisitors:  goalVisitors,
		goalEvents:    goalEvents,
		propVisitors:  propVisitors,
		propEvents:    propEvents,
	}
}

func (srv *MetricsServer) UpdateDataForSite(siteID string, data *plausible.TimeseriesData) {
	srv.pageviews.WithLabelValues(siteID).Set(float64(data.Pageviews))
	srv.visitors.WithLabelValues(siteID).Set(float64(data.Visitors))
	srv.bounceRate.WithLabelValues(siteID).Set(float64(data.BounceRate))
	srv.visitDuration.WithLabelValues(siteID).Set(float64(data.VisitDuration))
}

func (srv *MetricsServer) UpdateHealthStatusForSite(status map[string]bool) {
	for key, value := range status {
		if value {
			srv.healthStatus.WithLabelValues(key).Set(1)
		} else {
			srv.healthStatus.WithLabelValues(key).Set(0)
		}
	}
}

// UpdateGoalsForSite replaces all goal_* series for a site. The DeletePartialMatch
// keeps disappearing goals from sticking around as stale gauges.
func (srv *MetricsServer) UpdateGoalsForSite(siteID string, items []plausible.BreakdownItem) {
	match := prometheus.Labels{"site_id": siteID}
	srv.goalVisitors.DeletePartialMatch(match)
	srv.goalEvents.DeletePartialMatch(match)
	for _, item := range items {
		srv.goalVisitors.WithLabelValues(siteID, item.Name).Set(float64(item.Visitors))
		srv.goalEvents.WithLabelValues(siteID, item.Name).Set(float64(item.Events))
	}
}

// UpdatePropForSite replaces all prop_* series for (site, key). Same staleness
// reasoning as UpdateGoalsForSite — when a prop value drops out of the top-N
// the gauge for it must go too.
func (srv *MetricsServer) UpdatePropForSite(siteID, key string, items []plausible.BreakdownItem) {
	match := prometheus.Labels{"site_id": siteID, "key": key}
	srv.propVisitors.DeletePartialMatch(match)
	srv.propEvents.DeletePartialMatch(match)
	for _, item := range items {
		srv.propVisitors.WithLabelValues(siteID, key, item.Name).Set(float64(item.Visitors))
		srv.propEvents.WithLabelValues(siteID, key, item.Name).Set(float64(item.Events))
	}
}
