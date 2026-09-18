package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App            AppConfig              `yaml:"app"`
	Account        AccountConfig          `yaml:"account"`
	Feishu         FeishuConfig           `yaml:"feishu"`
	Schedule       ScheduleConfig         `yaml:"schedule"`
	ReportTemplate map[string]interface{} `yaml:"report_template"`
}

type AppConfig struct {
	Port     int    `yaml:"port"`
	Timezone string `yaml:"timezone"`
}

type AccountConfig struct {
	UserID      string `yaml:"user_id"`
	SystemToken string `yaml:"system_token"`
	APIBase     string `yaml:"api_base"`
}

type FeishuConfig struct {
	WebhookURL string `yaml:"webhook_url"`
	RetryTimes int    `yaml:"retry_times"`
}

type ScheduleConfig struct {
	SnapshotTime string `yaml:"snapshot_time"`
	ReportTime   string `yaml:"report_time"`
}

func Default() *Config {
	return &Config{
		App:      AppConfig{Port: 8080, Timezone: "Asia/Shanghai"},
		Account:  AccountConfig{APIBase: "https://api.quickrouter.ai"},
		Feishu:   FeishuConfig{RetryTimes: 3},
		Schedule: ScheduleConfig{SnapshotTime: "00:00", ReportTime: "10:30"},
	}
}

// Load 读取 YAML，不存在则生成默认配置，再应用环境变量覆盖
func Load(path string) (*Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	} else {
		out, _ := yaml.Marshal(cfg)
		if err := os.WriteFile(path, out, 0644); err != nil {
			return nil, err
		}
	}
	cfg.ApplyEnv()
	return cfg, nil
}

func (c *Config) ApplyEnv() {
	if v := os.Getenv("QR_USER_ID"); v != "" {
		c.Account.UserID = v
	}
	if v := os.Getenv("QR_SYSTEM_TOKEN"); v != "" {
		c.Account.SystemToken = v
	}
	if v := os.Getenv("QR_FEISHU_WEBHOOK"); v != "" {
		c.Feishu.WebhookURL = v
	}
}

// Save 写回 YAML（Web 设置页保存时调用）
func (c *Config) Save(path string) error {
	out, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}

// EnvOverridden 返回被环境变量覆盖的字段名集合
func (c *Config) EnvOverridden() map[string]bool {
	m := map[string]bool{}
	if os.Getenv("QR_USER_ID") != "" {
		m["user_id"] = true
	}
	if os.Getenv("QR_SYSTEM_TOKEN") != "" {
		m["system_token"] = true
	}
	if os.Getenv("QR_FEISHU_WEBHOOK") != "" {
		m["webhook_url"] = true
	}
	return m
}
