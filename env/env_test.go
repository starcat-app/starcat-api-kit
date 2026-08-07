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

func TestIntBoolLookupCSV(t *testing.T) {
	t.Setenv("KIT_ENV_INT", "42")
	if got := env.Int("KIT_ENV_INT", 1); got != 42 {
		t.Fatalf("Int=%d", got)
	}
	if got := env.Int("KIT_ENV_INT_MISSING", 7); got != 7 {
		t.Fatalf("Int fallback=%d", got)
	}
	t.Setenv("KIT_ENV_BOOL", "true")
	if !env.Bool("KIT_ENV_BOOL", false) {
		t.Fatal("Bool want true")
	}
	if env.Bool("KIT_ENV_BOOL_MISSING", true) != true {
		t.Fatal("Bool fallback")
	}
	t.Setenv("KIT_ENV_CSV", "x, y")
	if got := env.LookupCSV("KIT_ENV_CSV"); len(got) != 2 || got[0] != "x" {
		t.Fatalf("LookupCSV=%v", got)
	}
	if got := env.LookupCSV("KIT_ENV_CSV_EMPTY"); got != nil && len(got) != 0 {
		t.Fatalf("empty LookupCSV=%v", got)
	}
}
