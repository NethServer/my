/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package differ

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DifferConfig represents the complete differ configuration
type DifferConfig struct {
	Categorization CategorizationConfig `yaml:"categorization"`
	Severity       SeverityConfig       `yaml:"severity"`
	Significance   SignificanceConfig   `yaml:"significance"`
	Limits         LimitsConfig         `yaml:"limits"`
	Trends         TrendsConfig         `yaml:"trends"`
	Notifications  NotificationsConfig  `yaml:"notifications"`
}

// unifiedConfig represents the top-level config.yml structure
// The differ section is nested under the "differ" key
type unifiedConfig struct {
	Differ DifferConfig `yaml:"differ"`
}

// CategorizationConfig defines how fields are categorized
type CategorizationConfig struct {
	Categories map[string]CategoryRule `yaml:",inline"`
	Default    DefaultCategory         `yaml:"default"`
}

// CategoryRule defines patterns for a specific category
type CategoryRule struct {
	Patterns    []string `yaml:"patterns"`
	Description string   `yaml:"description"`
}

// DefaultCategory defines the default category for unmatched patterns
type DefaultCategory struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// SeverityConfig defines how severity is determined
type SeverityConfig struct {
	Critical SeverityLevel   `yaml:"critical"`
	High     SeverityLevel   `yaml:"high"`
	Medium   SeverityLevel   `yaml:"medium"`
	Low      SeverityLevel   `yaml:"low"`
	Default  DefaultSeverity `yaml:"default"`
}

// SeverityLevel defines conditions for a severity level
type SeverityLevel struct {
	Conditions  []SeverityCondition `yaml:"conditions"`
	Description string              `yaml:"description"`
}

// SeverityCondition defines a condition for severity matching
type SeverityCondition struct {
	ChangeType string   `yaml:"change_type"`
	Patterns   []string `yaml:"patterns"`
}

// DefaultSeverity defines the default severity
type DefaultSeverity struct {
	Level       string `yaml:"level"`
	Description string `yaml:"description"`
}

// SignificanceConfig defines what changes are significant
type SignificanceConfig struct {
	AlwaysSignificant []string            `yaml:"always_significant"`
	NeverSignificant  []string            `yaml:"never_significant"`
	TimeFilters       TimeFiltersConfig   `yaml:"time_filters"`
	ValueFilters      ValueFiltersConfig  `yaml:"value_filters"`
	Default           DefaultSignificance `yaml:"default"`
}

// TimeFiltersConfig defines time-based significance filters
type TimeFiltersConfig struct {
	IgnoreFrequent []TimeFilter `yaml:"ignore_frequent"`
}

// TimeFilter defines a time-based filter
type TimeFilter struct {
	Pattern       string `yaml:"pattern"`
	WindowSeconds int    `yaml:"window_seconds"`
}

// ValueFiltersConfig defines value-based significance filters
type ValueFiltersConfig struct {
	IgnoreMinor []ValueFilter `yaml:"ignore_minor"`
}

// ValueFilter defines a value-based filter
type ValueFilter struct {
	Pattern          string  `yaml:"pattern"`
	ThresholdPercent float64 `yaml:"threshold_percent"`
}

// DefaultSignificance defines default significance
type DefaultSignificance struct {
	Significant bool   `yaml:"significant"`
	Description string `yaml:"description"`
}

// LimitsConfig defines processing limits
type LimitsConfig struct {
	MaxDiffDepth       int `yaml:"max_diff_depth"`
	MaxDiffsPerRun     int `yaml:"max_diffs_per_run"`
	MaxFieldPathLength int `yaml:"max_field_path_length"`
}

// TrendsConfig defines trend analysis configuration
type TrendsConfig struct {
	Enabled        bool `yaml:"enabled"`
	WindowHours    int  `yaml:"window_hours"`
	MinOccurrences int  `yaml:"min_occurrences"`
}

// NotificationsConfig defines notification configuration
type NotificationsConfig struct {
	Grouping     GroupingConfig     `yaml:"grouping"`
	RateLimiting RateLimitingConfig `yaml:"rate_limiting"`
}

// GroupingConfig defines notification grouping
type GroupingConfig struct {
	Enabled           bool `yaml:"enabled"`
	TimeWindowMinutes int  `yaml:"time_window_minutes"`
	MaxGroupSize      int  `yaml:"max_group_size"`
}

// RateLimitingConfig defines rate limiting
type RateLimitingConfig struct {
	Enabled                 bool `yaml:"enabled"`
	MaxNotificationsPerHour int  `yaml:"max_notifications_per_hour"`
	MaxCriticalPerHour      int  `yaml:"max_critical_per_hour"`
}

// ConfigurableDiffer holds the configuration and compiled patterns
type ConfigurableDiffer struct {
	config               *DifferConfig
	categoryPatterns     map[string][]*regexp.Regexp
	severityPatterns     map[string]map[string][]*regexp.Regexp
	significancePatterns map[string]*regexp.Regexp
	loadTime             time.Time
}

// LoadConfig loads the differ configuration from YAML file
// Supports the unified config.yml format where differ config is nested under the "differ" key
func LoadConfig(configPath string) (*DifferConfig, error) {
	var data []byte

	if configPath != "" {
		// Explicit path provided
		var err error
		data, err = os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}
	} else {
		// Search for config.yml in standard locations
		searchPaths := []string{
			"config.yml",
			"../config.yml",
			"/etc/collect/config.yml",
		}

		// Add path relative to executable
		if execPath, err := os.Executable(); err == nil {
			searchPaths = append(searchPaths, filepath.Join(filepath.Dir(execPath), "config.yml"))
		}

		for _, path := range searchPaths {
			if d, err := os.ReadFile(path); err == nil {
				data = d
				break
			}
		}

		if data == nil {
			return nil, fmt.Errorf("config.yml not found in any of the search paths")
		}
	}

	// Try parsing as unified config (with "differ:" key)
	var unified unifiedConfig
	if err := yaml.Unmarshal(data, &unified); err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %w", err)
	}

	config := unified.Differ

	// If unified parsing resulted in empty config, try direct format
	if config.Limits.MaxDiffDepth == 0 && config.Limits.MaxDiffsPerRun == 0 {
		var direct DifferConfig
		if err := yaml.Unmarshal(data, &direct); err != nil {
			return nil, fmt.Errorf("failed to parse config YAML: %w", err)
		}
		config = direct
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// NewConfigurableDiffer creates a new differ with configuration
func NewConfigurableDiffer(configPath string) (*ConfigurableDiffer, error) {
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	differ := &ConfigurableDiffer{
		config:               config,
		categoryPatterns:     make(map[string][]*regexp.Regexp),
		severityPatterns:     make(map[string]map[string][]*regexp.Regexp),
		significancePatterns: make(map[string]*regexp.Regexp),
		loadTime:             time.Now(),
	}

	// Compile regex patterns for performance
	if err := differ.compilePatterns(); err != nil {
		return nil, fmt.Errorf("failed to compile patterns: %w", err)
	}

	return differ, nil
}

// compilePatterns compiles all regex patterns for performance
func (cd *ConfigurableDiffer) compilePatterns() error {
	// Compile category patterns
	for category, rule := range cd.config.Categorization.Categories {
		var patterns []*regexp.Regexp
		for _, pattern := range rule.Patterns {
			regex, err := regexp.Compile(strings.ToLower(pattern))
			if err != nil {
				return fmt.Errorf("invalid category pattern '%s' for category '%s': %w", pattern, category, err)
			}
			patterns = append(patterns, regex)
		}
		cd.categoryPatterns[category] = patterns
	}

	// Compile severity patterns
	severityLevels := map[string]SeverityLevel{
		"critical": cd.config.Severity.Critical,
		"high":     cd.config.Severity.High,
		"medium":   cd.config.Severity.Medium,
		"low":      cd.config.Severity.Low,
	}

	for level, severityLevel := range severityLevels {
		cd.severityPatterns[level] = make(map[string][]*regexp.Regexp)
		for _, condition := range severityLevel.Conditions {
			var patterns []*regexp.Regexp
			for _, pattern := range condition.Patterns {
				regex, err := regexp.Compile(strings.ToLower(pattern))
				if err != nil {
					return fmt.Errorf("invalid severity pattern '%s' for level '%s': %w", pattern, level, err)
				}
				patterns = append(patterns, regex)
			}
			cd.severityPatterns[level][condition.ChangeType] = patterns
		}
	}

	// Compile significance patterns
	for _, pattern := range cd.config.Significance.AlwaysSignificant {
		regex, err := regexp.Compile(strings.ToLower(pattern))
		if err != nil {
			return fmt.Errorf("invalid always significant pattern '%s': %w", pattern, err)
		}
		cd.significancePatterns["always_"+pattern] = regex
	}

	for _, pattern := range cd.config.Significance.NeverSignificant {
		regex, err := regexp.Compile(strings.ToLower(pattern))
		if err != nil {
			return fmt.Errorf("invalid never significant pattern '%s': %w", pattern, err)
		}
		cd.significancePatterns["never_"+pattern] = regex
	}

	return nil
}

// validateConfig validates the configuration
func validateConfig(config *DifferConfig) error {
	// Validate limits
	if config.Limits.MaxDiffDepth <= 0 {
		return fmt.Errorf("max_diff_depth must be positive")
	}
	if config.Limits.MaxDiffsPerRun <= 0 {
		return fmt.Errorf("max_diffs_per_run must be positive")
	}
	if config.Limits.MaxFieldPathLength <= 0 {
		return fmt.Errorf("max_field_path_length must be positive")
	}

	// Validate trends
	if config.Trends.WindowHours <= 0 {
		return fmt.Errorf("trends window_hours must be positive")
	}
	if config.Trends.MinOccurrences <= 0 {
		return fmt.Errorf("trends min_occurrences must be positive")
	}

	return nil
}

// GetConfig returns the current configuration
func (cd *ConfigurableDiffer) GetConfig() *DifferConfig {
	return cd.config
}

// GetLoadTime returns when the configuration was loaded
func (cd *ConfigurableDiffer) GetLoadTime() time.Time {
	return cd.loadTime
}

// ReloadConfig reloads the configuration from file
func (cd *ConfigurableDiffer) ReloadConfig(configPath string) error {
	config, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	cd.config = config
	cd.loadTime = time.Now()

	// Recompile patterns
	cd.categoryPatterns = make(map[string][]*regexp.Regexp)
	cd.severityPatterns = make(map[string]map[string][]*regexp.Regexp)
	cd.significancePatterns = make(map[string]*regexp.Regexp)

	return cd.compilePatterns()
}
