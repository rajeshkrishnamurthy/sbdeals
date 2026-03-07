package config

import "testing"

func TestDatabaseURLFromEnvSuccess(t *testing.T) {
	url, err := DatabaseURLFromEnv(func(key string) string {
		if key == "DATABASE_URL" {
			return "postgres://user:pass@localhost:5432/sbd?sslmode=disable"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if url == "" {
		t.Fatalf("expected non-empty database url")
	}
}

func TestDatabaseURLFromEnvMissing(t *testing.T) {
	_, err := DatabaseURLFromEnv(func(string) string { return "" })
	if err == nil {
		t.Fatalf("expected an error for missing DATABASE_URL")
	}
}
