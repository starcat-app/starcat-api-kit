package env_test

import (
	"os"
	"testing"
	"time"

	"github.com/starcat-app/starcat-api-kit/env"
)

func TestLookupRequired(t *testing.T) {
	t.Setenv("KIT_ENV_TEST_KEY", "value")
	got, err := env.LookupRequired("KIT_ENV_TEST_KEY")
	if err != nil || got != "value" {
		t.Fatalf("got %q err=%v", got, err)
	}
	_ = os.Unsetenv("KIT_ENV_TEST_MISSING")
	if _, err := env.LookupRequired("KIT_ENV_TEST_MISSING"); err == nil {
		t.Fatal("expected error")
	}
}

func TestCSVAndDuration(t *testing.T) {
	if got := env.CSV(" a, ,b "); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("CSV=%v", got)
	}
	t.Setenv("KIT_ENV_DUR", "5")
	if got := env.DurationSeconds("KIT_ENV_DUR", time.Second); got != 5*time.Second {
		t.Fatalf("duration=%v", got)
	}
}
