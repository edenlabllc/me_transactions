package cfg

import (
	"flag"

	"github.com/caarlos0/env"
)

type Config struct {
	MongoURL       string `env:"MONGO_URL"`
	DBWriteConcern int    `env:"DB_WRITE_CONCERN" envDefault:"1"`
	DBPoolSize     uint64 `env:"DB_POOL_SIZE" envDefault:"50"`

	GenServerName          string `env:"GEN_SERVER_NAME"`
	NodeName               string `env:"NODE_NAME"`
	AuditLogCollectionName string `env:"AUDIT_LOG_COLLECTION"`
	AuditLogEnabled        bool   `env:"AUDIT_LOG_ENABLED" envDefault:"true"`
	ErlangCookie           string `env:"ERLANG_COOKIE"`
	Port                   int    `env:"EMPD_PORT" envDefault:"15151"`
	LogLevel               string `env:"LOG_LEVEL"`
	HealthCheckPath        string `env:"HEALTH_CHECK_PATH"`
}

//
// ConfigFromEnv func - reads env by struct's fields 'env' annotation
//
func ConfigFromEnv() (*Config, error) {
	c := &Config{}
	if err := env.Parse(c); err != nil {
		return nil, err
	}
	cfgFromFlags(c)
	return c, nil
}

func cfgFromFlags(cfg *Config) {
	if cfg.MongoURL == "" {
		flag.StringVar(&cfg.MongoURL, "mongo_url", "mongodb://localhost:27017/medical_events?replicaSet=replicaTest", "mongo connect url")
	}

	if cfg.HealthCheckPath == "" {
		flag.StringVar(&cfg.HealthCheckPath, "health_check", "/tmp/healthy", "health check path")
	}

	if cfg.GenServerName == "" {
		flag.StringVar(&cfg.GenServerName, "gen_server", "mongo_transaction", "gen_server name")
	}

	if cfg.NodeName == "" {
		flag.StringVar(&cfg.NodeName, "name", "examplenode@127.0.0.1", "node name")
	}

	if cfg.AuditLogCollectionName == "" {
		flag.StringVar(&cfg.AuditLogCollectionName, "audit_log_collection", "audit_log", "audit log collection name")
	}

	if cfg.ErlangCookie == "" {
		flag.StringVar(&cfg.ErlangCookie, "cookie", "123", "cookie for interaction with erlang cluster")
	}

	if cfg.Port == 15151 {
		flag.IntVar(&cfg.Port, "epmd_port", 15151, "epmd port")
	}

	if cfg.LogLevel == "" {
		flag.StringVar(&cfg.LogLevel, "log_level", "info", "log level")
	}
}
