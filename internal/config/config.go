package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	appDirName       = "Tidyyy"
	configFileName   = "config.json"
	defaultMaxWords  = 5
	minimumNameWords = 2
	maximumNameWords = 5
)

type AppConfig struct {
	WatchDirs    []string `json:"watch_dirs"`
	CloudEnabled bool     `json:"cloud_enabled"`
	CloudAPIKey  string   `json:"cloud_api_key"`
	MaxNameWords int      `json:"max_name_words"`
}

func Default(homeDir string) AppConfig {
	watchDir := filepath.Join(homeDir, "Downloads")
	return AppConfig{
		WatchDirs:    []string{watchDir},
		CloudEnabled: false,
		CloudAPIKey:  "",
		MaxNameWords: defaultMaxWords,
	}
}

func Load() (AppConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return AppConfig{}, err
	}

	cfg := Default(home)
	path, err := path()
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	normalize(&cfg)
	return cfg, nil
}

func Save(cfg AppConfig) error {
	normalize(&cfg)
	path, err := path()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, payload, 0o600)
}

func path() (string, error) {
	base, err := os.UserConfigDir()
	fmt.Println("base: ", base)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appDirName, configFileName), nil
}

func normalize(cfg *AppConfig) {
	cfg.CloudAPIKey = strings.TrimSpace(cfg.CloudAPIKey)

	if cfg.MaxNameWords < minimumNameWords {
		cfg.MaxNameWords = minimumNameWords
	}
	if cfg.MaxNameWords > maximumNameWords {
		cfg.MaxNameWords = maximumNameWords
	}

	seen := map[string]struct{}{}
	cleaned := make([]string, 0, len(cfg.WatchDirs))
	for _, raw := range cfg.WatchDirs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		abs, err := filepath.Abs(raw)
		if err != nil {
			continue
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		cleaned = append(cleaned, abs)
	}
	fmt.Println(cfg)
	cfg.WatchDirs = cleaned
}
