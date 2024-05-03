package config

import "time"

type Config struct {
	Clients  ClientConfig   `toml:"clients"`
	Global   GlobalConfig   `toml:"global"`
	Log      LogConfig      `toml:"log"`
	Servers  ServersConfig  `toml:"servers"`
	Sentry   SentryConfig   `toml:"sentry"`
	DB       DBConfig       `toml:"db"`
	Services ServicesConfig `toml:"services"`
}

type ClientConfig struct {
	Keycloak Keycloak `toml:"keycloak" validate:"required"`
}

type Keycloak struct {
	BasePath     string `toml:"base_path" validate:"required"`
	Realm        string `toml:"realm" validate:"required"`
	ClientID     string `toml:"client_id" validate:"required"`
	ClientSecret string `toml:"client_secret" validate:"required"`
	DebugMode    bool   `toml:"debug_mode"`
}

type GlobalConfig struct {
	Env string `toml:"env" validate:"required,oneof=dev stage prod"`
}

type LogConfig struct {
	Level string `toml:"level" validate:"required,oneof=debug info warn error"`
}

type ServersConfig struct {
	Client ClientServerConfig `toml:"client"`
	Debug  DebugServerConfig  `toml:"debug"`
}

type ClientServerConfig struct {
	Addr           string         `toml:"addr" validate:"required,hostname_port"`
	AllowOrigins   []string       `toml:"allow_origins" validate:"dive,required,url"`
	RequiredAccess RequiredAccess `toml:"required_access" validate:"required"`
}

type RequiredAccess struct {
	Resource string `toml:"resource" validate:"required"`
	Role     string `toml:"role" validate:"required"`
}

type DebugServerConfig struct {
	Addr string `toml:"addr" validate:"required,hostname_port"`
}

type SentryConfig struct {
	DSN string `toml:"dsn"`
}

type DBConfig struct {
	Postgres PostgresConfig `toml:"postgres" validate:"required"`
}

type PostgresConfig struct {
	User      string `toml:"user" validate:"required"`
	Password  string `toml:"password" validate:"required"`
	Addr      string `toml:"addr" validate:"required,hostname_port"`
	Database  string `toml:"database" validate:"required"`
	DebugMode bool   `toml:"debug_mode"`
}

type ServicesConfig struct {
	MsgProducerConfig MsgProducerConfig `toml:"msg_producer"`
	OutboxConfig      OutboxConfig      `toml:"outbox"`
}

type MsgProducerConfig struct {
	Brokers    []string `toml:"brokers" validate:"dive,required,hostname_port"`
	Topic      string   `toml:"topic" validate:"required"`
	BatchSize  int      `toml:"batch_size" validate:"required,gte=1"`
	EncryptKey string   `toml:"encrypt_key"`
}

type OutboxConfig struct {
	Workers    int           `toml:"workers" validate:"required,gte=1"`
	IdleTime   time.Duration `toml:"idle_time" validate:"required"`
	ReserveFor time.Duration `toml:"reserve_for" validate:"required"`
}

func (c Config) IsProduction() bool {
	return c.Global.Env == "prod"
}
