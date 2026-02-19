package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/riesinger/plausible-exporter/plausible"
)

type MetricsServer struct {
	pageviews     *prometheus.GaugeVec
	visitors      *prometheus.GaugeVec
	bounceRate    *prometheus.GaugeVec
	visitDuration *prometheus.GaugeVec
	healthStatus  *prometheus.GaugeVec
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
		Name:      "health_status",
		Help:      "Health status of the Plausible API (1 for healthy, 0 for unhealthy)",
	}, []string{"component"})

	return &MetricsServer{
		pageviews:     pageviews,
		visitors:      visitors,
		bounceRate:    bounceRate,
		visitDuration: visitDuration,
		healthStatus:  healthStatus,
	}
}

func (srv *MetricsServer) UpdateDataForSite(siteID string, data *plausible.TimeseriesData) {
	srv.pageviews.WithLabelValues(siteID).Set(float64(data.Pageviews))
	srv.visitors.WithLabelValues(siteID).Set(float64(data.Visitors))
	srv.bounceRate.WithLabelValues(siteID).Set(float64(data.BounceRate))
	srv.visitDuration.WithLabelValues(siteID).Set(float64(data.VisitDuration))
}

func (srv *MetricsServer) UpdateHealthStatusForSite(status *map[string]bool) {
	for key, value := range *status {
		if value {
			srv.healthStatus.WithLabelValues(key).Set(1)
		} else {
			srv.healthStatus.WithLabelValues(key).Set(0)
		}
	}
}
