package config

import (
	"os"
)

func ConfigSetup() {
	// Настройки DB
	os.Setenv("DB_USERNAME", "postgres")
	os.Setenv("DB_PASSWORD", "S8859306s")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_NAME", "NATS1")

	os.Setenv("DB_POOL_MAXCONN", "5")
	os.Setenv("DB_POOL_MAXCONN_LIFETIME", "300")

	// Настройки NATS-Streaming
	os.Setenv("NATS_HOSTS", "localhost:4222")
	os.Setenv("NATS_CLUSTER_ID", "test-cluster")
	os.Setenv("NATS_CLIENT_ID", "sub-4")
	os.Setenv("NATS_SUBJECT", "delapaska1")
	os.Setenv("NATS_DURABLE_NAME", "Replica-1")
	os.Setenv("NATS_ACK_WAIT_SECONDS", "30")

	// Настройки Cache
	os.Setenv("CACHE_SIZE", "10")
	os.Setenv("APP_KEY", "WB-1")
}
