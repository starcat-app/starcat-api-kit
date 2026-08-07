package github_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/starcat-app/starcat-api-kit/github"
	"github.com/starcat-app/starcat-api-kit/tokenpool"
)

func TestGetRepoOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/octo/hello" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer tok_test_1234567890" {
			t.Fatalf("auth=%q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":               42,
			"name":             "hello",
			"full_name":        "octo/hello",
			"stargazers_count": 9,
			"forks_count":      1,
			"owner":            map[string]any{"login": "octo"},
		})
	}))
	defer srv.Close()

	pool := tokenpool.New([]string{"tok_test_1234567890"})
	client := github.NewClient(github.Options{
		BaseURL: srv.URL,
		Pool:    pool,
		Limiter: github.NewRateLimitHandler(time.Millisecond),
	})
	repo, err := client.GetRepo(context.Background(), "octo", "hello")
	if err != nil {
		t.Fatal(err)
	}
	if repo.ID != 42 || repo.Owner != "octo" || repo.Stars != 9 {
		t.Fatalf("repo=%+v", repo)
	}
}

func TestGetRepoNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	client := github.NewClient(github.Options{
		BaseURL: srv.URL,
		Pool:    tokenpool.New([]string{"tok_test_1234567890"}),
	})
	_, err := client.GetRepo(context.Background(), "a", "b")
	if !errors.Is(err, github.ErrRepoNotFound) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetRepoRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	client := github.NewClient(github.Options{
		BaseURL: srv.URL,
		Pool:    tokenpool.New([]string{"tok_test_1234567890"}),
	})
	// DisableUntil 最少约 60s；用短 ctx 避免单测卡住，只要首次路径返回限流或 ctx 取消即可。
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, err := client.GetRepo(ctx, "a", "b")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, github.ErrRateLimited) && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetRepoAnonymousOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Fatalf("anonymous request should not send Authorization, got %q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":               1,
			"name":             "Starcat",
			"full_name":        "starcat-app/Starcat",
			"html_url":         "https://github.com/starcat-app/Starcat",
			"is_template":      false,
			"stargazers_count": 10,
			"owner":            map[string]any{"login": "starcat-app"},
		})
	}))
	defer srv.Close()

	client := github.NewClient(github.Options{
		BaseURL:        srv.URL,
		Pool:           tokenpool.New(nil),
		AllowAnonymous: true,
	})
	repo, err := client.GetRepo(context.Background(), "starcat-app", "Starcat")
	if err != nil {
		t.Fatal(err)
	}
	if repo.HTMLURL == "" || repo.Owner != "starcat-app" {
		t.Fatalf("repo=%+v", repo)
	}
}
