package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config is the TOML-backed runtime configuration surface for NVMS commands.
type Config struct {
	Name   string `toml:"name"`
	Image  string `toml:"image"`
	Tier   int    `toml:"tier"`
	CPU    int    `toml:"cpu"`
	Memory int    `toml:"memory"`
}

// Load reads, decodes, and validates a TOML configuration file.
func Load(path string) (*Config, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("config path is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	cfg, err := parse(data)
	if err != nil {
		return nil, fmt.Errorf("decode config %q: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config %q: %w", path, err)
	}

	return cfg, nil
}

// LoadFromArgs loads a config using the first positional CLI argument.
func LoadFromArgs() (*Config, error) {
	path, err := pathFromArgs(os.Args)
	if err != nil {
		return nil, err
	}

	return Load(path)
}

// Validate enforces the minimum required configuration surface.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(c.Image) == "" {
		return errors.New("image is required")
	}
	if c.Tier < 1 || c.Tier > 3 {
		return errors.New("tier must be between 1 and 3")
	}
	if c.CPU < 1 {
		return errors.New("cpu must be at least 1")
	}
	if c.Memory < 64 {
		return errors.New("memory must be at least 64")
	}

	return nil
}

func pathFromArgs(args []string) (string, error) {
	if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
		return "", errors.New("usage: <command> <config.toml>")
	}

	return args[1], nil
}

func parse(data []byte) (*Config, error) {
	var cfg Config

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			return nil, fmt.Errorf("line %d: tables are not supported", lineNo)
		}

		key, raw, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: expected key = value", lineNo)
		}

		key = strings.TrimSpace(key)
		raw = strings.TrimSpace(stripComment(raw))

		switch key {
		case "name":
			value, err := parseString(raw)
			if err != nil {
				return nil, fmt.Errorf("line %d: parse name: %w", lineNo, err)
			}
			cfg.Name = value
		case "image":
			value, err := parseString(raw)
			if err != nil {
				return nil, fmt.Errorf("line %d: parse image: %w", lineNo, err)
			}
			cfg.Image = value
		case "tier":
			value, err := strconv.Atoi(raw)
			if err != nil {
				return nil, fmt.Errorf("line %d: parse tier: %w", lineNo, err)
			}
			cfg.Tier = value
		case "cpu":
			value, err := strconv.Atoi(raw)
			if err != nil {
				return nil, fmt.Errorf("line %d: parse cpu: %w", lineNo, err)
			}
			cfg.CPU = value
		case "memory":
			value, err := strconv.Atoi(raw)
			if err != nil {
				return nil, fmt.Errorf("line %d: parse memory: %w", lineNo, err)
			}
			cfg.Memory = value
		default:
			return nil, fmt.Errorf("line %d: unknown key %q", lineNo, key)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func parseString(raw string) (string, error) {
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return "", errors.New("expected quoted string")
	}

	value, err := strconv.Unquote(raw)
	if err != nil {
		return "", err
	}

	return value, nil
}

func stripComment(raw string) string {
	inQuotes := false
	for i := 0; i < len(raw); i++ {
		switch raw[i] {
		case '"':
			if i == 0 || raw[i-1] != '\\' {
				inQuotes = !inQuotes
			}
		case '#':
			if !inQuotes {
				return raw[:i]
			}
		}
	}

	return raw
}
