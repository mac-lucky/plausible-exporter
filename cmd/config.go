package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// PropConfig describes a single custom-property key to break down.
// Limit caps the number of values fetched per site (0 = API default 100, max 1000).
// Use it to bound cardinality on high-fanout keys like path, title, video_id.
type PropConfig struct {
	Key   string
	Limit int
}

var (
	listenAddress   string
	bearerAuthToken string
	plausibleHost   *url.URL
	token           string
	siteIDs         []string
	period          string
	goalsEnabled    bool
	goalsLimit      int
	propConfigs     []PropConfig
)

func readConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/plausible-exporter/")

	viper.SetDefault("listen_address", "0.0.0.0:8080")
	viper.SetDefault("period", "day")
	viper.SetDefault("goals_enabled", false)
	viper.SetDefault("goals_limit", 100)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("config: failed to read config file: %w", err)
		}
	}

	listenAddress = viper.GetString("listen_address")
	plausibleHostRaw := viper.GetString("plausible_host")
	if plausibleHostRaw == "" {
		return fmt.Errorf("config: no plausible host provided")
	}
	var err error
	plausibleHost, err = url.Parse(plausibleHostRaw)
	if err != nil {
		return fmt.Errorf("config: cannot parse plausible host as URL: %w", err)
	}
	token = viper.GetString("plausible_token")
	if token == "" {
		return fmt.Errorf("config: no plausible token provided")
	}
	viper.UnmarshalKey("plausible_site_ids", &siteIDs, viper.DecodeHook(mapstructure.StringToSliceHookFunc(",")))
	if len(siteIDs) == 0 {
		return fmt.Errorf("config: no plausible site IDs provided")
	}

	period = viper.GetString("period")
	goalsEnabled = viper.GetBool("goals_enabled")
	goalsLimit = viper.GetInt("goals_limit")

	propConfigs, err = parsePropKeys(viper.GetString("prop_keys"))
	if err != nil {
		return err
	}

	bearerAuthToken = viper.GetString("bearer_auth_token")

	return nil
}

// parsePropKeys accepts a comma-separated list. Each entry is either "key"
// (use default limit) or "key:limit" — e.g. "theme,viewport,path:20,title:50".
// Returns nil (with no error) for an empty string.
func parsePropKeys(raw string) ([]PropConfig, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]PropConfig, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		key, limStr, hasLimit := strings.Cut(p, ":")
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("config: empty prop key in entry %q", p)
		}
		cfg := PropConfig{Key: key}
		if hasLimit {
			lim, err := strconv.Atoi(strings.TrimSpace(limStr))
			if err != nil || lim < 0 {
				return nil, fmt.Errorf("config: invalid limit %q for prop key %q", limStr, key)
			}
			cfg.Limit = lim
		}
		out = append(out, cfg)
	}
	return out, nil
}
