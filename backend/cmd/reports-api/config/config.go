package config

import "os"

type Config struct {
	Addr           string
	ClickhouseURL  string
	ClickhouseUser string
	ClickhousePass string
	Issuer         string
	JwksURL        string
}

func Create() *Config {
	return &Config{
		Addr:           env("API_ADDR", ":8000"),
		ClickhouseURL:  env("CLICKHOUSE_URL", "http://clickhouse:8123"),
		ClickhouseUser: env("CLICKHOUSE_USER", "default"),
		ClickhousePass: env("CLICKHOUSE_PASSWORD", "clickhouse_password"),
		Issuer:         env("KEYCLOAK_ISSUER", "http://localhost:8080/realms/reports-realm"),
		JwksURL:        env("KEYCLOAK_JWKS_URL", "http://keycloak:8080/realms/reports-realm/protocol/openid-connect/certs"),
	}
}

func env(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
