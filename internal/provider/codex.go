package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"

	"ratelimitbar/internal/usage"
)

// DefaultCodexTimeout は、Fetch 1回分(起動から応答まで)の上限。
const DefaultCodexTimeout = 20 * time.Second

// CodexProcess は起動済みの codex app-server。Stop は子プロセスを終了させ、待ち合わせる。
type CodexProcess struct {
	Stdin  io.WriteCloser
	Stdout io.Reader
	Stop   func() error
}

// CodexLauncher は ctx に紐づく codex app-server を起動する。
type CodexLauncher func(ctx context.Context) (CodexProcess, error)

type codex struct {
	launch  CodexLauncher
	timeout time.Duration
}

// NewCodex は、PATH 上の codex を Fetch ごとに起動する Provider を返す。
func NewCodex() Provider {
	return NewCodexWithPath(nil)
}

func NewCodexWithPath(path func() string) Provider {
	return NewCodexWithLauncher(func(ctx context.Context) (CodexProcess, error) {
		configured := ""
		if path != nil {
			configured = path()
		}
		executable, err := ResolveExecutable("codex", configured)
		if err != nil {
			return CodexProcess{}, err
		}
		return launchCodex(ctx, string(executable))
	}, DefaultCodexTimeout)
}

func NewCodexWithLauncher(launch CodexLauncher, timeout time.Duration) Provider {
	return codex{launch: launch, timeout: timeout}
}

func (codex) Name() string { return "Codex" }

func launchCodex(ctx context.Context, path string) (CodexProcess, error) {
	cmd := exec.CommandContext(ctx, path, "app-server")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return CodexProcess{}, fmt.Errorf("codex stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return CodexProcess{}, fmt.Errorf("codex stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return CodexProcess{}, fmt.Errorf("start codex: %w", err)
	}
	return CodexProcess{
		Stdin:  stdin,
		Stdout: stdout,
		Stop: func() error {
			stdin.Close()
			cmd.Process.Kill()
			cmd.Wait()
			return nil
		},
	}, nil
}

type rpcRequest struct {
	ID     *int   `json:"id,omitempty"`
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

type rpcResponse struct {
	ID     json.RawMessage `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

const (
	initializeRequestID = 1
	rateLimitsRequestID = 2
)

func (c codex) Fetch(ctx context.Context) (usage.Usage, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	proc, err := c.launch(ctx)
	if err != nil {
		return usage.Usage{}, err
	}
	defer proc.Stop()

	lines := readLines(ctx, proc.Stdout)
	send := func(req rpcRequest) error {
		body, err := json.Marshal(req)
		if err != nil {
			return err
		}
		_, err = proc.Stdin.Write(append(body, '\n'))
		return err
	}

	id := initializeRequestID
	clientInfo := map[string]any{"clientInfo": map[string]string{"name": "ratelimitbar", "version": "0"}}
	if err := send(rpcRequest{ID: &id, Method: "initialize", Params: clientInfo}); err != nil {
		return usage.Usage{}, fmt.Errorf("send codex initialize: %w", err)
	}
	if _, err := awaitResponse(ctx, lines, initializeRequestID); err != nil {
		return usage.Usage{}, fmt.Errorf("codex initialize: %w", err)
	}
	if err := send(rpcRequest{Method: "initialized"}); err != nil {
		return usage.Usage{}, fmt.Errorf("send codex initialized: %w", err)
	}
	id = rateLimitsRequestID
	if err := send(rpcRequest{ID: &id, Method: "account/rateLimits/read"}); err != nil {
		return usage.Usage{}, fmt.Errorf("send codex rate limits request: %w", err)
	}
	result, err := awaitResponse(ctx, lines, rateLimitsRequestID)
	if err != nil {
		return usage.Usage{}, fmt.Errorf("codex rate limits: %w", err)
	}
	windows, err := parseCodexRateLimits(result)
	if err != nil {
		return usage.Usage{}, err
	}
	return usage.Usage{Windows: windows}, nil
}

// readLines は r を行単位で読んで返す。r が閉じるか ctx が終了すると、channel を閉じる。
func readLines(ctx context.Context, r io.Reader) <-chan []byte {
	lines := make(chan []byte)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(r)
		scanner.Buffer(nil, 1<<20)
		for scanner.Scan() {
			line := append([]byte(nil), scanner.Bytes()...)
			select {
			case lines <- line:
			case <-ctx.Done():
				return
			}
		}
	}()
	return lines
}

// awaitResponse は、id が wantID の応答を待つ。id のない通知や、別 id の応答は読み飛ばす。
func awaitResponse(ctx context.Context, lines <-chan []byte, wantID int) (json.RawMessage, error) {
	want := fmt.Sprint(wantID)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case line, ok := <-lines:
			if !ok {
				return nil, errors.New("codex app-server closed before responding")
			}
			var resp rpcResponse
			if json.Unmarshal(line, &resp) != nil || string(resp.ID) != want {
				continue
			}
			if resp.Error != nil {
				return nil, errors.New(resp.Error.Message)
			}
			return resp.Result, nil
		}
	}
}
