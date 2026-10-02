package provider_test

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

	"ratelimitbar/internal/provider"
	"ratelimitbar/internal/usage"
)

// cursorToken は、実際の access token と同じ JWT の形をした偽の値。sub から session cookie の userId を作る。
var cursorToken = fakeJWT("user_test", "SECRET-SIGNATURE-0123456789")

func fakeJWT(userID, signature string) string {
	enc := base64.RawURLEncoding.EncodeToString
	return enc([]byte(`{"alg":"none"}`)) + "." + enc([]byte(`{"sub":"auth0|`+userID+`"}`)) + "." + signature
}

// cursorUsageBody は usage API の応答の例。実機で応答形式を確定した後、その形式に合わせて更新する。
const cursorUsageBody = `{"individualUsage":{"plan":{"used":1800,"limit":10000,"totalPercentUsed":18}}}`

type fakeAuthStore struct {
	mu     sync.Mutex
	tokens []string
	err    error
	calls  int
}

func (f *fakeAuthStore) AccessToken(context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	token := f.tokens[0]
	if len(f.tokens) > 1 {
		f.tokens = f.tokens[1:]
	}
	return token, nil
}

type fakeHTTP struct {
	do       func(*http.Request) (*http.Response, error)
	mu       sync.Mutex
	requests []*http.Request
}

func (f *fakeHTTP) Do(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	f.requests = append(f.requests, r)
	f.mu.Unlock()
	return f.do(r)
}

func respondWith(status int, body string) func(*http.Request) (*http.Response, error) {
	return func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
	}
}

func requestCarriesToken(r *http.Request, token string) bool {
	if strings.Contains(r.URL.String(), token) {
		return true
	}
	for _, values := range r.Header {
		for _, v := range values {
			if strings.Contains(v, token) {
				return true
			}
		}
	}
	return false
}

func TestCursorNameIsCursor(t *testing.T) {
	p := provider.NewCursorWith(&fakeAuthStore{tokens: []string{cursorToken}}, &fakeHTTP{do: respondWith(200, cursorUsageBody)})
	if got := p.Name(); got != "Cursor" {
		t.Errorf("Name() = %q, want Cursor", got)
	}
}

func TestCursorFetchReturnsOnlyMonthlyWindow(t *testing.T) {
	p := provider.NewCursorWith(&fakeAuthStore{tokens: []string{cursorToken}}, &fakeHTTP{do: respondWith(200, cursorUsageBody)})

	got, err := p.Fetch(context.Background())

	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	want := []usage.Window{{Label: "Monthly", UsedPercent: 18}}
	if !slices.Equal(got.Windows, want) {
		t.Errorf("Windows = %v, want %v", got.Windows, want)
	}
	if got.Stale {
		t.Error("Stale = true, want false for live data")
	}
}

func TestCursorFetchSendsTheTokenOnlyToACursorHostOverHTTPS(t *testing.T) {
	client := &fakeHTTP{do: respondWith(200, cursorUsageBody)}
	p := provider.NewCursorWith(&fakeAuthStore{tokens: []string{cursorToken}}, client)

	if _, err := p.Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(client.requests))
	}
	r := client.requests[0]
	host := r.URL.Hostname()
	if r.URL.Scheme != "https" || (host != "cursor.com" && !strings.HasSuffix(host, ".cursor.com") && !strings.HasSuffix(host, ".cursor.sh")) {
		t.Errorf("request URL = %s, want https on a Cursor host", r.URL.Redacted())
	}
	if !requestCarriesToken(r, cursorToken) {
		t.Error("request does not carry the access token")
	}
}

func TestCursorFetchSendsTheSessionCookieBuiltFromTheTokenSubject(t *testing.T) {
	client := &fakeHTTP{do: respondWith(200, cursorUsageBody)}
	p := provider.NewCursorWith(&fakeAuthStore{tokens: []string{cursorToken}}, client)

	if _, err := p.Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}

	want := "WorkosCursorSessionToken=user_test%3A%3A" + cursorToken
	if got := client.requests[0].Header.Get("Cookie"); got != want {
		t.Errorf("Cookie = %q, want %q", got, want)
	}
}

func TestCursorFetchFailsWhenTotalPercentUsedIsMissing(t *testing.T) {
	bodies := map[string]string{
		"empty plan":        `{"individualUsage":{"plan":{}}}`,
		"null totalPercent": `{"individualUsage":{"plan":{"totalPercentUsed":null}}}`,
		"empty object":      `{}`,
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			p := provider.NewCursorWith(&fakeAuthStore{tokens: []string{cursorToken}}, &fakeHTTP{do: respondWith(200, body)})

			got, err := p.Fetch(context.Background())

			if err == nil {
				t.Fatal("Fetch() error = nil, want error")
			}
			if len(got.Windows) != 0 {
				t.Errorf("Windows = %v, want none", got.Windows)
			}
		})
	}
}

func TestCursorFetchFailsWithoutCallingTheAPIWhenTheTokenIsNotAJWT(t *testing.T) {
	client := &fakeHTTP{do: respondWith(200, cursorUsageBody)}
	p := provider.NewCursorWith(&fakeAuthStore{tokens: []string{"not-a-jwt"}}, client)

	_, err := p.Fetch(context.Background())

	if err == nil {
		t.Fatal("Fetch() error = nil, want error")
	}
	if len(client.requests) != 0 {
		t.Errorf("requests = %d, want 0", len(client.requests))
	}
}

func TestCursorFetchReadsTheTokenFromTheAuthStoreOnEveryFetch(t *testing.T) {
	auth := &fakeAuthStore{tokens: []string{fakeJWT("u1", "first-aaaa"), fakeJWT("u2", "second-bbbb")}}
	client := &fakeHTTP{do: respondWith(200, cursorUsageBody)}
	p := provider.NewCursorWith(auth, client)

	for range 2 {
		if _, err := p.Fetch(context.Background()); err != nil {
			t.Fatal(err)
		}
	}

	if auth.calls != 2 {
		t.Errorf("AuthStore calls = %d, want 2 (token must not be kept between fetches)", auth.calls)
	}
	if !requestCarriesToken(client.requests[0], fakeJWT("u1", "first-aaaa")) || !requestCarriesToken(client.requests[1], fakeJWT("u2", "second-bbbb")) {
		t.Error("each request must carry the token read for that fetch")
	}
}

func TestCursorFetchReturnsAuthStoreErrorWithoutCallingTheAPI(t *testing.T) {
	authErr := errors.New("keychain item not found")
	client := &fakeHTTP{do: respondWith(200, cursorUsageBody)}
	p := provider.NewCursorWith(&fakeAuthStore{err: authErr}, client)

	_, err := p.Fetch(context.Background())

	if !errors.Is(err, authErr) {
		t.Errorf("Fetch() error = %v, want it to wrap %v", err, authErr)
	}
	if len(client.requests) != 0 {
		t.Errorf("requests = %d, want 0 without a token", len(client.requests))
	}
}

func TestCursorFetchErrorsNeverContainTheToken(t *testing.T) {
	tests := map[string]*fakeHTTP{
		"transport error that echoes the token": {do: func(*http.Request) (*http.Response, error) {
			return nil, errors.New("dial failed for Cookie: " + cursorToken)
		}},
		"unauthorized response that echoes the token": {do: respondWith(401, `{"error":"bad token `+cursorToken+`"}`)},
		"server error response":                       {do: respondWith(500, "token "+cursorToken)},
		"malformed body that echoes the token":        {do: respondWith(200, "not json "+cursorToken)},
	}
	for name, client := range tests {
		t.Run(name, func(t *testing.T) {
			p := provider.NewCursorWith(&fakeAuthStore{tokens: []string{cursorToken}}, client)

			_, err := p.Fetch(context.Background())

			if err == nil {
				t.Fatal("Fetch() error = nil, want error")
			}
			if strings.Contains(err.Error(), cursorToken) {
				t.Errorf("error %q contains the access token", err)
			}
		})
	}
}
