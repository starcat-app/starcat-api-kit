package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/starcat-app/starcat-api-kit/tokenpool"
)

// 哨兵错误：调用方可用 errors.Is 判断。
var (
	ErrRepoNotFound = errors.New("repo not found (404)")
	ErrRateLimited  = errors.New("rate limited")
)

// Repo 是 GET /repos/{owner}/{repo} 的中立 DTO（字段并集）。
type Repo struct {
	ID            int64
	Owner         string
	Name          string
	FullName      string
	Description   *string
	Homepage      *string
	Language      *string
	Stars         int
	Forks         int
	Watchers      int
	Subscribers   int
	OpenIssues    int
	Topics        []string
	LicenseSpdx   *string
	OwnerAvatar   *string
	Archived      bool
	Fork          bool
	Private       bool
	DefaultBranch string
	PushedAt      string
	UpdatedAt     string
	CreatedAt     string
}

// HTTPError 非 200/404/429 的 GitHub HTTP 错误。
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string { return e.Message }

// Client 统一 GitHub REST 客户端。
type Client struct {
	baseURL   string
	userAgent string
	http      *http.Client
	pool      *tokenpool.Pool
	limiter   *RateLimitHandler
}

// Options 控制 Client 装配。
type Options struct {
	BaseURL    string
	UserAgent  string
	HTTPClient *http.Client
	Pool       *tokenpool.Pool
	Limiter    *RateLimitHandler
	Timeout    time.Duration
}

// NewClient 创建客户端。Pool 必填（可含空 token 列表，但 GetRepo 需要可用 token）。
func NewClient(opt Options) *Client {
	if opt.Pool == nil {
		opt.Pool = tokenpool.New(nil)
	}
	timeout := opt.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	httpClient := opt.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}
	baseURL := strings.TrimRight(strings.TrimSpace(opt.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	ua := strings.TrimSpace(opt.UserAgent)
	if ua == "" {
		ua = "starcat-api-kit"
	}
	return &Client{
		baseURL:   baseURL,
		userAgent: ua,
		http:      httpClient,
		pool:      opt.Pool,
		limiter:   opt.Limiter,
	}
}

// SetBaseURL 覆盖 API 基础 URL（测试用）。
func (c *Client) SetBaseURL(url string) { c.baseURL = strings.TrimRight(url, "/") }

// GetRepo 调 GET /repos/{owner}/{repo}，最多重试 3 次。
func (c *Client) GetRepo(ctx context.Context, owner, repo string) (*Repo, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := c.getRepoOnce(ctx, owner, repo)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if errors.Is(err, ErrRateLimited) {
			continue
		}
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && (httpErr.StatusCode == http.StatusUnauthorized || httpErr.StatusCode >= 500) {
			continue
		}
		return nil, err
	}
	return nil, fmt.Errorf("GetRepo %s/%s failed after 3 attempts: %w", owner, repo, lastErr)
}

func (c *Client) getRepoOnce(ctx context.Context, owner, repo string) (*Repo, error) {
	token := c.pool.PickBest()
	if token == nil {
		resetAt := c.pool.EarliestReset()
		if !resetAt.IsZero() && resetAt.After(time.Now()) {
			d := time.Until(resetAt)
			log.Printf("[github] no tokens, sleeping %v until %s", d.Round(time.Second), resetAt.Format(time.RFC3339))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(d):
			}
		}
		return nil, ErrRateLimited
	}

	if c.limiter != nil {
		c.limiter.Wait()
	}

	url := fmt.Sprintf("%s/repos/%s/%s", c.baseURL, owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req, token.Value)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	c.pool.UpdateFromResponse(token, resp)

	switch resp.StatusCode {
	case http.StatusOK:
		var apiResp repoAPIResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return nil, fmt.Errorf("decode repo response: %w", err)
		}
		return apiResp.toRepo(), nil
	case http.StatusNotFound:
		return nil, ErrRepoNotFound
	case http.StatusForbidden, http.StatusTooManyRequests:
		c.handleRateLimit(resp, token)
		return nil, ErrRateLimited
	case http.StatusUnauthorized:
		return nil, &HTTPError{StatusCode: resp.StatusCode, Message: "unauthorized"}
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		msg := fmt.Sprintf("GitHub /repos/%s/%s HTTP %d: %s", owner, repo, resp.StatusCode, strings.TrimSpace(string(body)))
		return nil, &HTTPError{StatusCode: resp.StatusCode, Message: msg}
	}
}

// GetReadme 调 GET /repos/{owner}/{repo}/readme，返回解码后的 Markdown 文本。
func (c *Client) GetReadme(ctx context.Context, owner, repo string) (string, error) {
	for attempt := 0; attempt < 3; attempt++ {
		content, err := c.getReadmeOnce(ctx, owner, repo)
		if err == nil {
			return content, nil
		}
		if errors.Is(err, ErrRateLimited) {
			continue
		}
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && (httpErr.StatusCode == http.StatusUnauthorized || httpErr.StatusCode >= 500) {
			continue
		}
		return "", err
	}
	return "", fmt.Errorf("GetReadme %s/%s failed after 3 attempts", owner, repo)
}

func (c *Client) getReadmeOnce(ctx context.Context, owner, repo string) (string, error) {
	token := c.pool.PickBest()
	if token == nil {
		return "", ErrRateLimited
	}
	if c.limiter != nil {
		c.limiter.Wait()
	}
	url := fmt.Sprintf("%s/repos/%s/%s/readme", c.baseURL, owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	c.setHeaders(req, token.Value)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	c.pool.UpdateFromResponse(token, resp)

	switch resp.StatusCode {
	case http.StatusOK:
		var readmeResp struct {
			Content  string `json:"content"`
			Encoding string `json:"encoding"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&readmeResp); err != nil {
			return "", fmt.Errorf("decode readme: %w", err)
		}
		if readmeResp.Encoding != "base64" {
			return "", fmt.Errorf("unsupported readme encoding: %s", readmeResp.Encoding)
		}
		cleaned := strings.Map(func(r rune) rune {
			if r == '\n' || r == '\r' || r == ' ' {
				return -1
			}
			return r
		}, readmeResp.Content)
		decoded, err := base64.StdEncoding.DecodeString(cleaned)
		if err != nil {
			return "", fmt.Errorf("decode README base64: %w", err)
		}
		return string(decoded), nil
	case http.StatusNotFound:
		return "", ErrRepoNotFound
	case http.StatusForbidden, http.StatusTooManyRequests:
		c.handleRateLimit(resp, token)
		return "", ErrRateLimited
	case http.StatusUnauthorized:
		return "", &HTTPError{StatusCode: resp.StatusCode, Message: "unauthorized"}
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", &HTTPError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("GitHub %s HTTP %d: %s", url, resp.StatusCode, strings.TrimSpace(string(body)))}
	}
}

func (c *Client) setHeaders(req *http.Request, token string) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", c.userAgent)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

func (c *Client) handleRateLimit(resp *http.Response, token *tokenpool.TokenState) {
	pauseUntil := token.ResetAt
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		if secs, err := strconv.Atoi(retryAfter); err == nil && secs > 0 {
			ra := time.Now().Add(time.Duration(secs) * time.Second)
			if ra.After(pauseUntil) {
				pauseUntil = ra
			}
		}
	}
	if pauseUntil.Before(time.Now().Add(60 * time.Second)) {
		pauseUntil = time.Now().Add(60 * time.Second)
	}
	log.Printf("[github] rate limited (%d), disabling token until %s", resp.StatusCode, pauseUntil.Format(time.RFC3339))
	c.pool.DisableUntil(token, pauseUntil, fmt.Sprintf("rate limited status %d", resp.StatusCode))
}

type repoAPIResponse struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	FullName      string   `json:"full_name"`
	Description   *string  `json:"description"`
	Homepage      *string  `json:"homepage"`
	Language      *string  `json:"language"`
	Stars         int      `json:"stargazers_count"`
	Forks         int      `json:"forks_count"`
	Watchers      int      `json:"watchers_count"`
	Subscribers   int      `json:"subscribers_count"`
	OpenIssues    int      `json:"open_issues_count"`
	Topics        []string `json:"topics"`
	Archived      bool     `json:"archived"`
	Fork          bool     `json:"fork"`
	Private       bool     `json:"private"`
	DefaultBranch string   `json:"default_branch"`
	PushedAt      string   `json:"pushed_at"`
	UpdatedAt     string   `json:"updated_at"`
	CreatedAt     string   `json:"created_at"`
	License       *struct {
		SpdxID *string `json:"spdx_id"`
	} `json:"license"`
	Owner *struct {
		Login     string  `json:"login"`
		AvatarURL *string `json:"avatar_url"`
	} `json:"owner"`
}

func (r *repoAPIResponse) toRepo() *Repo {
	out := &Repo{
		ID:            r.ID,
		Name:          r.Name,
		FullName:      r.FullName,
		Description:   r.Description,
		Homepage:      r.Homepage,
		Language:      r.Language,
		Stars:         r.Stars,
		Forks:         r.Forks,
		Watchers:      r.Watchers,
		Subscribers:   r.Subscribers,
		OpenIssues:    r.OpenIssues,
		Topics:        r.Topics,
		Archived:      r.Archived,
		Fork:          r.Fork,
		Private:       r.Private,
		DefaultBranch: r.DefaultBranch,
		PushedAt:      r.PushedAt,
		UpdatedAt:     r.UpdatedAt,
		CreatedAt:     r.CreatedAt,
	}
	if r.License != nil {
		out.LicenseSpdx = r.License.SpdxID
	}
	if r.Owner != nil {
		out.Owner = r.Owner.Login
		out.OwnerAvatar = r.Owner.AvatarURL
	}
	return out
}
