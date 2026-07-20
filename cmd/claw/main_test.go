package main

import (
	"strings"
	"testing"
)

func TestCheckRequiredEnvVars(t *testing.T) {
	keys := []string{"ZHIPU_API_KEY", "SLACK_BOT_TOKEN", "SLACK_APP_TOKEN"}
	for _, key := range keys {
		t.Run("缺少_"+key, func(t *testing.T) {
			for _, envKey := range keys {
				t.Setenv(envKey, "test-value")
			}
			t.Setenv(key, "")

			err := checkRequiredEnvVars()
			if err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("checkRequiredEnvVars() error = %v, want 包含 %s", err, key)
			}
		})
	}
}

func TestCheckRequiredEnvVarsAcceptsCompleteConfig(t *testing.T) {
	t.Setenv("ZHIPU_API_KEY", "model-key")
	t.Setenv("SLACK_BOT_TOKEN", "xoxb-test")
	t.Setenv("SLACK_APP_TOKEN", "xapp-test")

	if err := checkRequiredEnvVars(); err != nil {
		t.Fatalf("checkRequiredEnvVars() error = %v", err)
	}
}
