package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	URL      string `yaml:"url"`
	Enabled  bool   `yaml:"enabled"`
	Token    string `yaml:"token"`
	Username string `yaml:"username"`
	LastSync int64  `yaml:"last_sync"`
}

type Config struct {
	DBPath           string        `yaml:"db_path"`
	EditorMode       string        `yaml:"editor_mode"`
	Theme            string        `yaml:"theme"`
	AutoSaveInterval time.Duration `yaml:"auto_save_interval"`
	Salt             string        `yaml:"salt"`
	Language         string        `yaml:"language"`
	Server           ServerConfig  `yaml:"server"`
}

func DefaultConfigPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "config.yml"
	}
	return filepath.Join(filepath.Dir(exe), "config.yml")
}

func DefaultDBPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "jotaku.db"
	}
	return filepath.Join(filepath.Dir(exe), "jotaku.db")
}

func ConfigExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		DBPath:           DefaultDBPath(),
		EditorMode:       "normal",
		Theme:            "dark",
		AutoSaveInterval: 3 * time.Second,
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.DBPath == "" {
		cfg.DBPath = DefaultDBPath()
	}

	if cfg.DBPath[0] == '~' {
		home, _ := os.UserHomeDir()
		cfg.DBPath = filepath.Join(home, cfg.DBPath[1:])
	}

	return cfg, nil
}

func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

func (c *Config) GetSalt() ([]byte, error) {
	if c.Salt == "" {
		return nil, nil
	}
	return base64.StdEncoding.DecodeString(c.Salt)
}

func (c *Config) SetSalt(salt []byte) {
	c.Salt = base64.StdEncoding.EncodeToString(salt)
}
