package cfg

import (
	"fmt"
)

type Config struct {
	MongodbHost     string `env:"MONGODB_HOST,required"`
	MongodbPort     string `env:"MONGODB_PORT,required"`
	MongodbUsername string `env:"MONGODB_USERNAME,required"`
	MongodbPassword string `env:"MONGODB_PASSWORD,required"`
	MongoDBName     string `env:"MONGODB_NAME_USER,required"`

	DBWriteConcern int `env:"DB_WRITE_CONCERN"`

	AuditLogEnabled bool `env:"AUDIT_LOG_ENABLED" envDefault:"10"`
}

//
// GetMongodbDSN method - generate mongodb dsn string
//
func (c *Config) GetMongodbDSN() string {
	return fmt.Sprintf("mongodb://%s:%s/audit_log?replicaSet=rs0", c.MongodbHost, c.MongodbPort)
}

//
// ConfigFromEnv func - reads env by struct's fields 'env' annotation
//
func ConfigFromEnv() (*Config, error) {
	c := &Config{
		MongodbHost:     "localhost",
		MongodbPort:     "27017",
		MongodbUsername: "root",
		MongodbPassword: "password",
		MongoDBName:     "audit_log",
		AuditLogEnabled: true,
	}
	//if err := env.Parse(c); err != nil {
	//	return nil, err
	//}
	return c, nil
}
