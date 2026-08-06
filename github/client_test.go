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
	_, err := client.GetRepo(context.Background(), "a", "b")
	if !errors.Is(err, github.ErrRateLimited) {
		t.Fatalf("err=%v", err)
	}
}
