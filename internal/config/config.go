package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env, HTTPAddr, AdminAddr, TelegramToken, PostgresAddr, RedisAddr, QueueMode, RedisQueueKey, StorageDir, AdminToken, AudDToken string
	WorkerConcurrency                                                                                                             int
	MaxFileSizeMB                                                                                                                 int64
	JobTimeout                                                                                                                    time.Duration
	RateLimitPerMinute                                                                                                            int
	MaxAttempts                                                                                                                   int
	CleanupAge                                                                                                                    time.Duration
	RetryBase                                                                                                                     time.Duration
	TelegramMaxUploadMB                                                                                                           int64
}

func Load() Config {
	return Config{Env: get("MORIS_ENV", "development"), HTTPAddr: get("MORIS_HTTP_ADDR", ":8080"), AdminAddr: get("MORIS_ADMIN_ADDR", ":8081"), TelegramToken: os.Getenv("MORIS_TELEGRAM_TOKEN"), PostgresAddr: get("MORIS_POSTGRES_ADDR", "postgres:5432"), RedisAddr: get("MORIS_REDIS_ADDR", "redis:6379"), QueueMode: get("MORIS_QUEUE_MODE", "redis"), RedisQueueKey: get("MORIS_REDIS_QUEUE_KEY", "moris:queue:default"), StorageDir: get("MORIS_STORAGE_DIR", "./data/moris"), WorkerConcurrency: getInt("MORIS_WORKER_CONCURRENCY", 2), MaxFileSizeMB: int64(getInt("MORIS_MAX_FILE_SIZE_MB", 2048)), JobTimeout: getDuration("MORIS_JOB_TIMEOUT", 30*time.Minute), RateLimitPerMinute: getInt("MORIS_RATE_LIMIT_PER_MINUTE", 20), AdminToken: os.Getenv("MORIS_ADMIN_TOKEN"), AudDToken: os.Getenv("MORIS_AUDD_TOKEN"), MaxAttempts: getInt("MORIS_MAX_ATTEMPTS", 3), CleanupAge: getDuration("MORIS_CLEANUP_AGE", 24*time.Hour), RetryBase: getDuration("MORIS_RETRY_BASE", 5*time.Second), TelegramMaxUploadMB: int64(getInt("MORIS_TELEGRAM_MAX_UPLOAD_MB", 49))}
}
func get(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func getInt(k string, d int) int {
	v, e := strconv.Atoi(os.Getenv(k))
	if e != nil || v <= 0 {
		return d
	}
	return v
}
func getDuration(k string, d time.Duration) time.Duration {
	v, e := time.ParseDuration(os.Getenv(k))
	if e != nil || v <= 0 {
		return d
	}
	return v
}
