// Package config 负责加载并校验 new-api-proxy 的运行配置。
//
// 加载优先级（高→低）：环境变量 > config.yaml > 默认值。
// yaml 文本中的 ${ENV} 占位符会先展开，再被环境变量覆盖。
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultPort         = 8080
	defaultLogLevel     = "info"
	defaultArkBaseURL   = "https://ark.cn-beijing.volces.com/api/v3"
	defaultArkTimeout   = 120 * time.Second
	defaultReadTimeout  = 60 * time.Second
	defaultWriteTimeout = 300 * time.Second // 视频/ASR 轮询需较长写出超时
	defaultSpeechTimeout = 60 * time.Second
)

// Config 全局配置根。
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Proxy    ProxyConfig    `yaml:"proxy"`
	Ark      ArkConfig      `yaml:"ark"`
	Speech   SpeechConfig   `yaml:"speech"`
	ModelMap map[string]string `yaml:"model_map"` // OpenAI 模型名 -> 火山 Model ID
	VoiceMap map[string]string `yaml:"voice_map"` // OpenAI voice -> 火山 voice_type
}

// ServerConfig HTTP 服务配置。
type ServerConfig struct {
	Port            int    `yaml:"port"`
	ReadTimeoutStr  string `yaml:"read_timeout"`
	WriteTimeoutStr string `yaml:"write_timeout"`
	LogLevel        string `yaml:"log_level"`

	ReadTimeout  time.Duration `yaml:"-"`
	WriteTimeout time.Duration `yaml:"-"`
}

// ProxyConfig 本代理入口配置。
type ProxyConfig struct {
	// AuthKey 入口鉴权 Key；留空表示不校验（适合内网，置于 new-api 之后）。
	AuthKey string `yaml:"auth_key"`
}

// ArkConfig 火山方舟配置（文本/嵌入/图片/视频）。
type ArkConfig struct {
	APIKey     string        `yaml:"api_key"`
	BaseURL    string        `yaml:"base_url"`
	Region     string        `yaml:"region"`
	TimeoutStr string        `yaml:"timeout"`
	Timeout    time.Duration `yaml:"-"`
}

// SpeechConfig 火山「语音技术」产品线配置（TTS/ASR，独立鉴权）。
type SpeechConfig struct {
	AppID        string        `yaml:"app_id"`
	AccessToken  string        `yaml:"access_token"`
	TTSURL       string        `yaml:"tts_url"`
	ASRSubmitURL string        `yaml:"asr_submit_url"`
	ASRQueryURL  string        `yaml:"asr_query_url"`
	TimeoutStr   string        `yaml:"timeout"`
	Timeout      time.Duration `yaml:"-"`
}

// Load 从 path 读取配置；文件不存在时仅使用默认值与环境变量。
// path 为空时跳过文件读取。
func Load(path string) (*Config, error) {
	cfg := defaultConfig()

	if path != "" {
		if data, err := os.ReadFile(path); err == nil {
			// 先展开 ${ENV} 占位符，再解析 yaml。
			expanded := os.ExpandEnv(string(data))
			if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
				return nil, fmt.Errorf("parse config %s: %w", path, err)
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read config %s: %w", path, err)
		}
	}

	cfg.applyEnv()

	if err := cfg.parseDurations(); err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            defaultPort,
			LogLevel:        defaultLogLevel,
			ReadTimeoutStr:  defaultReadTimeout.String(),
			WriteTimeoutStr: defaultWriteTimeout.String(),
		},
		Ark: ArkConfig{
			BaseURL:    defaultArkBaseURL,
			Region:     "cn-beijing",
			TimeoutStr: defaultArkTimeout.String(),
		},
		Speech: SpeechConfig{
			TimeoutStr: defaultSpeechTimeout.String(),
		},
		ModelMap: map[string]string{},
		VoiceMap: map[string]string{},
	}
}

// applyEnv 用环境变量覆盖关键标量（大写 + 下划线）。
func (c *Config) applyEnv() {
	envInt("SERVER_PORT", func(v int) { c.Server.Port = v })
	envStr("SERVER_LOG_LEVEL", func(v string) { c.Server.LogLevel = v })
	envStr("SERVER_READ_TIMEOUT", func(v string) { c.Server.ReadTimeoutStr = v })
	envStr("SERVER_WRITE_TIMEOUT", func(v string) { c.Server.WriteTimeoutStr = v })

	envStr("PROXY_AUTH_KEY", func(v string) { c.Proxy.AuthKey = v })

	envStr("ARK_API_KEY", func(v string) { c.Ark.APIKey = v })
	envStr("ARK_BASE_URL", func(v string) { c.Ark.BaseURL = v })

	envStr("SPEECH_APP_ID", func(v string) { c.Speech.AppID = v })
	envStr("SPEECH_ACCESS_TOKEN", func(v string) { c.Speech.AccessToken = v })
	envStr("SPEECH_TTS_URL", func(v string) { c.Speech.TTSURL = v })
	envStr("SPEECH_ASR_SUBMIT_URL", func(v string) { c.Speech.ASRSubmitURL = v })
	envStr("SPEECH_ASR_QUERY_URL", func(v string) { c.Speech.ASRQueryURL = v })
}

func envStr(name string, set func(string)) {
	if v, ok := os.LookupEnv(name); ok && v != "" {
		set(v)
	}
}

func envInt(name string, set func(int)) {
	if v, ok := os.LookupEnv(name); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			set(n)
		}
	}
}

// parseDurations 将字符串超时配置解析为 time.Duration。
func (c *Config) parseDurations() error {
	var err error
	if c.Server.ReadTimeout, err = parseDuration(c.Server.ReadTimeoutStr, defaultReadTimeout); err != nil {
		return err
	}
	if c.Server.WriteTimeout, err = parseDuration(c.Server.WriteTimeoutStr, defaultWriteTimeout); err != nil {
		return err
	}
	if c.Ark.Timeout, err = parseDuration(c.Ark.TimeoutStr, defaultArkTimeout); err != nil {
		return err
	}
	if c.Speech.Timeout, err = parseDuration(c.Speech.TimeoutStr, defaultSpeechTimeout); err != nil {
		return err
	}
	return nil
}

func parseDuration(s string, fallback time.Duration) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", s, err)
	}
	return d, nil
}

func (c *Config) validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server.port: %d", c.Server.Port)
	}
	if c.Ark.BaseURL == "" {
		c.Ark.BaseURL = defaultArkBaseURL
	}
	c.Ark.BaseURL = strings.TrimRight(c.Ark.BaseURL, "/")
	return nil
}

// ResolveModel 将 OpenAI 模型名解析为火山 Model ID；未配置映射则原样返回（透传）。
func (c *Config) ResolveModel(name string) string {
	if name == "" {
		return name
	}
	if v, ok := c.ModelMap[name]; ok && v != "" {
		return v
	}
	return name
}

// ResolveVoice 将 OpenAI voice 名解析为火山 voice_type；未配置则原样返回。
func (c *Config) ResolveVoice(name string) string {
	if name == "" {
		return name
	}
	if v, ok := c.VoiceMap[name]; ok && v != "" {
		return v
	}
	return name
}

// LogLevel 返回 slog 级别（默认 info）。
func (c *Config) LogLevel() string {
	if c.Server.LogLevel == "" {
		return defaultLogLevel
	}
	return strings.ToLower(c.Server.LogLevel)
}
