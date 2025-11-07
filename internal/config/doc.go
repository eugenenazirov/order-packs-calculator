// Package config loads runtime configuration from multiple sources (YAML files,
// environment variables, CLI flags) with precedence: CLI flags > YAML config >
// Environment variables > Defaults. It exposes strongly typed settings to the
// rest of the application.
package config
