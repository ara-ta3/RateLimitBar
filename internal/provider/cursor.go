package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"ratelimitbar/internal/usage"
)

const (
	cursorUsageURL         = "https://cursor.com/api/usage-summary"
	cursorKeychainService  = "cursor-access-token"
	cursorRequestTimeout   = 15 * time.Second
	cursorMaxResponseBytes = 1 << 20
)

// CursorAuthStore は Cursor のローカル認証情報から access token を取り出す。
type CursorAuthStore interface {
	AccessToken(ctx context.Context) (string, error)
}

// CursorUsageClient は usage API へ request を送る HTTP 部。
type CursorUsageClient interface {
	Do(*http.Request) (*http.Response, error)
}

type cursor struct {
	auth   CursorAuthStore
	client CursorUsageClient
}

type cursorUsageSummary struct {
	BillingCycleEnd time.Time `json:"billingCycleEnd"`
	IndividualUsage struct {
		Plan struct {
			TotalPercentUsed *float64 `json:"totalPercentUsed"`
		} `json:"plan"`
	} `json:"individualUsage"`
}

// NewCursor は、macOS Keychain の Cursor の access token で usage API を呼ぶ Provider を返す。
func NewCursor() Provider {
	return NewCursorWith(keychainAuthStore{}, &http.Client{Timeout: cursorRequestTimeout})
}

func NewCursorWith(auth CursorAuthStore, client CursorUsageClient) Provider {
	return cursor{auth: auth, client: client}
}

func (cursor) Name() string { return "Cursor" }

// Fetch は token を呼び出しごとに取得し、保持しない。error には token を含めない。
func (c cursor) Fetch(ctx context.Context) (usage.Usage, error) {
	token, err := c.auth.AccessToken(ctx)
	if err != nil {
		return usage.Usage{}, fmt.Errorf("read cursor access token: %w", err)
	}
	req, err := newCursorUsageRequest(ctx, token)
	if err != nil {
		return usage.Usage{}, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return usage.Usage{}, fmt.Errorf("request cursor usage: %s", redact(err.Error(), token))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return usage.Usage{}, fmt.Errorf("cursor usage API returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, cursorMaxResponseBytes))
	if err != nil {
		return usage.Usage{}, fmt.Errorf("read cursor usage response: %s", redact(err.Error(), token))
	}
	var summary cursorUsageSummary
	if err := json.Unmarshal(body, &summary); err != nil {
		return usage.Usage{}, errors.New("parse cursor usage response: invalid JSON")
	}
	percent := summary.IndividualUsage.Plan.TotalPercentUsed
	if percent == nil {
		return usage.Usage{}, errors.New("cursor usage response has no individualUsage.plan.totalPercentUsed")
	}
	return usage.Usage{Windows: []usage.Window{{Label: WindowMonthly, UsedPercent: int(math.Round(*percent)), ResetsAt: summary.BillingCycleEnd}}}, nil
}

// newCursorUsageRequest は、Cursor の Web と同じ session cookie(<userId>::<access token>)で request を作る。
// userId は access token(JWT)の sub から得る。
func newCursorUsageRequest(ctx context.Context, token string) (*http.Request, error) {
	userID, err := cursorUserID(token)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cursorUsageURL, nil)
	if err != nil {
		return nil, errors.New("build cursor usage request")
	}
	req.Header.Set("Cookie", "WorkosCursorSessionToken="+userID+"%3A%3A"+token)
	return req, nil
}

func cursorUserID(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", errors.New("cursor access token is not a JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return "", errors.New("cursor access token payload is not base64")
	}
	var claims struct {
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Sub == "" {
		return "", errors.New("cursor access token has no sub claim")
	}
	return claims.Sub[strings.LastIndex(claims.Sub, "|")+1:], nil
}

func redact(message, token string) string {
	return strings.ReplaceAll(message, token, "[redacted]")
}

// keychainAuthStore は macOS の login Keychain から Cursor が保存した token を、都度読み出す。
type keychainAuthStore struct{}

func (keychainAuthStore) AccessToken(ctx context.Context) (string, error) {
	var stdout bytes.Buffer
	cmd := exec.CommandContext(ctx, "security", "find-generic-password", "-s", cursorKeychainService, "-w")
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("cursor token not found in keychain: %w", err)
	}
	token := strings.TrimSpace(stdout.String())
	if token == "" {
		return "", errors.New("cursor token in keychain is empty")
	}
	return token, nil
}
