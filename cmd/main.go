package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mac-lucky/plausible-exporter/plausible"
	"github.com/mac-lucky/plausible-exporter/prometheus"
	"github.com/mac-lucky/plausible-exporter/server"
)

func main() {
	if err := readConfig(); err != nil {
		log.Fatal(err)
	}

	log.Println("Starting plausible-exporter")

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	plausibleClients := make(map[string]*plausible.Client)
	for _, siteID := range siteIDs {
		plausibleClients[siteID] = &plausible.Client{HostAPIBase: plausibleHost, SiteID: siteID, Token: token}
	}
	plausibleApp := &plausible.Client{HostAPIBase: plausibleHost}

	metrics := prometheus.NewServer(siteIDs)

	updatePlausibleData := func() {
		for _, siteID := range siteIDs {
			clt := plausibleClients[siteID]
			data, err := clt.GetTimeseriesData()
			if err != nil {
				log.Printf("Refreshing data for site %s failed: %v", siteID, err)
				continue
			}
			metrics.UpdateDataForSite(siteID, data)

			if goalsEnabled {
				if items, err := clt.GetBreakdown("event:goal", "goal", period, goalsLimit); err != nil {
					log.Printf("Refreshing goals for site %s failed: %v", siteID, err)
				} else {
					metrics.UpdateGoalsForSite(siteID, items)
				}
			}

			for _, prop := range propConfigs {
				property := "event:props:" + prop.Key
				items, err := clt.GetBreakdown(property, prop.Key, period, prop.Limit)
				if err != nil {
					log.Printf("Refreshing prop %q for site %s failed: %v", prop.Key, siteID, err)
					continue
				}
				metrics.UpdatePropForSite(siteID, prop.Key, items)
			}

			log.Printf("Data for site %s was refreshed from plausible", siteID)
		}

		status, err := plausibleApp.GetHealth()
		if err != nil {
			log.Printf("Refreshing health status failed: %v", err)
			return
		}
		metrics.UpdateHealthStatusForSite(status)
	}

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		updatePlausibleData()
		for {
			select {
			case <-ticker.C:
				updatePlausibleData()
			case <-ctx.Done():
				log.Println("Stopping plausible refresh timer")
				return
			}
		}
	}()

	srv := server.New()
	if bearerAuthToken != "" {
		srv.SetBearerAuthToken(bearerAuthToken)
	}

	go func() {
		if err := srv.ListenAndServe(listenAddress); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v\n", err)
		}
	}()

	log.Println("Server started")
	<-ctx.Done()
	log.Println("Shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Graceful shutdown failed: %v\n", err)
	}
	log.Println("Bye")
}
