package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	TestID   string          `yaml:"test_id"`
	Target   Target          `yaml:"target"`
	Scale    Scale           `yaml:"scale"`
	MNOPools map[string]MNO  `yaml:"mno_pools"`
	Journeys []Journey       `yaml:"journeys"`
	Behavior Behavior        `yaml:"behavior"`
	Abort    AbortConditions `yaml:"abort_conditions"`
	Seed     int64           `yaml:"seed"`
}

type Target struct {
	SenderURL    string `yaml:"sender_url"`
	APIKeyHeader string `yaml:"api_key_header"`
	APIKey       string `yaml:"api_key"`
	DryRun       bool   `yaml:"dry_run"`
}

type Scale struct {
	TotalRequestsTarget      int `yaml:"total_requests_target"`
	ConcurrentSessionsTarget int `yaml:"concurrent_sessions_target"`
	WorkerID                 int `yaml:"worker_id"`
	WorkerCount              int `yaml:"worker_count"`
}

type MNO struct {
	Prefixes []string `yaml:"prefixes"`
	Weight   float64  `yaml:"weight"`
}

type Journey struct {
	Name   string  `yaml:"name"`
	Weight float64 `yaml:"weight"`
	Steps  []Step  `yaml:"steps"`
}

type Step struct {
	USSD  string `yaml:"ussd"`
	Input string `yaml:"input"`
}

type Behavior struct {
	AbandonmentRate float64 `yaml:"abandonment_rate"`
	SessionTimeoutS int     `yaml:"session_timeout_s"`
	MockErrorRate   float64 `yaml:"mock_error_rate"`
}

type AbortConditions struct {
	ErrorRateCeiling float64 `yaml:"error_rate_ceiling"`
}

var envPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	data = envPattern.ReplaceAllFunc(data, func(value []byte) []byte {
		name := string(value[2 : len(value)-1])
		return []byte(os.Getenv(name))
	})
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Target.SenderURL) == "" {
		return fmt.Errorf("target.sender_url is required")
	}
	if c.Scale.WorkerCount < 0 || c.Scale.WorkerID < 0 || c.Scale.WorkerID >= maxInt(c.Scale.WorkerCount, 1) {
		return fmt.Errorf("scale.worker_id must be between 0 and scale.worker_count - 1")
	}
	if len(c.Journeys) == 0 {
		return fmt.Errorf("at least one journey is required")
	}
	if c.Behavior.MockErrorRate < 0 || c.Behavior.MockErrorRate > 1 {
		return fmt.Errorf("behavior.mock_error_rate must be between 0 and 1")
	}
	for _, journey := range c.Journeys {
		if journey.Name == "" || len(journey.Steps) == 0 || journey.Weight <= 0 {
			return fmt.Errorf("journey %q must have a name, positive weight, and steps", journey.Name)
		}
	}
	return nil
}

func maxInt(first, second int) int {
	if first > second {
		return first
	}
	return second
}
