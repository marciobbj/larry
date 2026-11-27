package plugin

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// LSPClient manages a single Language Server subprocess using JSON-RPC 2.0
// Content-Length framing over stdin/stdout.
type LSPClient struct {
	Name    string
	RootDir string

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser

	writeMu sync.Mutex
	mu      sync.Mutex

	nextID   atomic.Int64
	pending  map[int64]func(result any, rpcErr any)
	notifs   map[string][]func(params any)
	onExit   []func(errMsg string)
	running  atomic.Bool
	closed   atomic.Bool
	dispatch func(AsyncEvent)
}

type lspMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

// StartLSPClient spawns an LSP server process and starts its background reader loop.
func StartLSPClient(
	name string,
	command string,
	args []string,
	rootDir string,
	env map[string]string,
	dispatch func(AsyncEvent),
) (*LSPClient, error) {
	if command == "" {
		return nil, fmt.Errorf("lsp command is required")
	}
	if name == "" {
		name = filepath.Base(command)
	}

	cmd := exec.Command(command, args...)
	if rootDir != "" {
		if info, err := os.Stat(rootDir); err == nil && info.IsDir() {
			cmd.Dir = rootDir
		}
	}
	if len(env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("lsp stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("lsp stdout pipe: %w", err)
	}
	// Discard stderr so LSP logs never corrupt Larry's TUI screen.
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start lsp %q: %w", command, err)
	}

	client := &LSPClient{
		Name:     name,
		RootDir:  rootDir,
		cmd:      cmd,
		stdin:    stdin,
		stdout:   stdout,
		pending:  make(map[int64]func(result any, rpcErr any)),
		notifs:   make(map[string][]func(params any)),
		dispatch: dispatch,
	}
	client.running.Store(true)

	go client.readLoop()
	return client, nil
}

// IsRunning reports whether the LSP process is still alive.
func (c *LSPClient) IsRunning() bool {
	return c.running.Load()
}

// Request sends a JSON-RPC request and registers a callback invoked on the UI thread.
func (c *LSPClient) Request(method string, params any, cb func(result any, rpcErr any)) (int64, error) {
	if !c.running.Load() {
		return 0, fmt.Errorf("lsp client %q is not running", c.Name)
	}
	id := c.nextID.Add(1)
	if cb != nil {
		c.mu.Lock()
		c.pending[id] = cb
		c.mu.Unlock()
	}

	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if params != nil {
		payload["params"] = params
	}
	if err := c.writeFrame(payload); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return 0, err
	}
	return id, nil
}

// Notify sends a JSON-RPC notification (no response expected).
func (c *LSPClient) Notify(method string, params any) error {
	if !c.running.Load() {
		return fmt.Errorf("lsp client %q is not running", c.Name)
	}
	payload := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if params != nil {
		payload["params"] = params
	}
	return c.writeFrame(payload)
}

// OnNotification registers a handler for a server notification method (or "*" for all).
func (c *LSPClient) OnNotification(method string, fn func(params any)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.notifs[method] = append(c.notifs[method], fn)
}

// OnExit registers a handler invoked when the LSP process exits.
func (c *LSPClient) OnExit(fn func(errMsg string)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onExit = append(c.onExit, fn)
}

// Stop gracefully closes stdin and terminates the LSP process.
func (c *LSPClient) Stop() {
	if !c.closed.CompareAndSwap(false, true) {
		return
	}
	c.running.Store(false)
	// Best-effort LSP exit notification before closing stdin.
	_ = c.writeFrame(map[string]any{"jsonrpc": "2.0", "method": "exit"})
	_ = c.stdin.Close()
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
}

func (c *LSPClient) writeFrame(payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))

	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if _, err := io.WriteString(c.stdin, header); err != nil {
		return err
	}
	_, err = c.stdin.Write(body)
	return err
}

func (c *LSPClient) readLoop() {
	reader := bufio.NewReaderSize(c.stdout, 64*1024)
	var readErr error

	for {
		contentLen, err := readLSPHeaders(reader)
		if err != nil {
			readErr = err
			break
		}
		if contentLen <= 0 {
			continue
		}

		body := make([]byte, contentLen)
		if _, err := io.ReadFull(reader, body); err != nil {
			readErr = err
			break
		}

		var msg lspMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			continue
		}

		c.handleIncoming(msg)
	}

	waitErr := c.cmd.Wait()
	c.running.Store(false)

	errMsg := ""
	if !c.closed.Load() {
		if waitErr != nil {
			errMsg = waitErr.Error()
		} else if readErr != nil && readErr != io.EOF {
			errMsg = readErr.Error()
		}
	}

	c.mu.Lock()
	exitHandlers := append([]func(string){}, c.onExit...)
	pending := c.pending
	c.pending = make(map[int64]func(any, any))
	c.mu.Unlock()

	if c.dispatch != nil {
		for _, cb := range pending {
			callback := cb
			c.dispatch(AsyncEvent{
				Kind: "lsp_response",
				Fn: func(_ EditorAPI) {
					callback(nil, map[string]any{"message": "lsp client exited"})
				},
			})
		}
		for _, h := range exitHandlers {
			handler := h
			msgCopy := errMsg
			c.dispatch(AsyncEvent{
				Kind: "lsp_exit",
				Fn: func(_ EditorAPI) {
					handler(msgCopy)
				},
			})
		}
	}
}

func (c *LSPClient) handleIncoming(msg lspMessage) {
	// Server-to-client request (has both ID and Method): respond with null result
	// so servers like gopls / lua-language-server don't stall on capability registration.
	if msg.ID != nil && msg.Method != "" {
		_ = c.writeFrame(map[string]any{
			"jsonrpc": "2.0",
			"id":      *msg.ID,
			"result":  nil,
		})
		return
	}

	// Response to one of our requests
	if msg.ID != nil && msg.Method == "" {
		c.mu.Lock()
		cb := c.pending[*msg.ID]
		delete(c.pending, *msg.ID)
		c.mu.Unlock()

		if cb == nil || c.dispatch == nil {
			return
		}

		var result any
		if len(msg.Result) > 0 && !bytes.Equal(bytes.TrimSpace(msg.Result), []byte("null")) {
			_ = json.Unmarshal(msg.Result, &result)
		}
		var rpcErr any
		if len(msg.Error) > 0 && !bytes.Equal(bytes.TrimSpace(msg.Error), []byte("null")) {
			_ = json.Unmarshal(msg.Error, &rpcErr)
		}

		c.dispatch(AsyncEvent{
			Kind: "lsp_response",
			Fn: func(_ EditorAPI) {
				cb(result, rpcErr)
			},
		})
		return
	}

	// Notification from server
	if msg.Method != "" {
		c.mu.Lock()
		handlers := append([]func(any){}, c.notifs[msg.Method]...)
		wildcard := append([]func(any){}, c.notifs["*"]...)
		c.mu.Unlock()

		if len(handlers) == 0 && len(wildcard) == 0 {
			return
		}

		var params any
		if len(msg.Params) > 0 {
			_ = json.Unmarshal(msg.Params, &params)
		}

		if c.dispatch != nil {
			for _, h := range handlers {
				handler := h
				p := params
				c.dispatch(AsyncEvent{
					Kind: "lsp_notification",
					Fn: func(_ EditorAPI) {
						handler(p)
					},
				})
			}
			for _, h := range wildcard {
				handler := h
				p := map[string]any{"method": msg.Method, "params": params}
				c.dispatch(AsyncEvent{
					Kind: "lsp_notification",
					Fn: func(_ EditorAPI) {
						handler(p)
					},
				})
			}
		}
	}
}

func readLSPHeaders(r *bufio.Reader) (int, error) {
	contentLen := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return 0, err
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if contentLen >= 0 {
				return contentLen, nil
			}
			continue
		}
		if colon := strings.IndexByte(trimmed, ':'); colon > 0 {
			key := strings.ToLower(strings.TrimSpace(trimmed[:colon]))
			val := strings.TrimSpace(trimmed[colon+1:])
			if key == "content-length" {
				n, err := strconv.Atoi(val)
				if err == nil {
					contentLen = n
				}
			}
		}
	}
}

// PathToURI converts a local filesystem path into a file:// URI suitable for LSP.
func PathToURI(path string) string {
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err == nil {
		path = abs
	}
	slashed := filepath.ToSlash(path)
	if !strings.HasPrefix(slashed, "/") {
		slashed = "/" + slashed
	}
	u := url.URL{Scheme: "file", Path: slashed}
	return u.String()
}

// URIToPath converts an LSP file:// URI back into a local filesystem path.
func URIToPath(rawURI string) string {
	if !strings.HasPrefix(rawURI, "file://") {
		return rawURI
	}
	u, err := url.Parse(rawURI)
	if err != nil {
		return strings.TrimPrefix(rawURI, "file://")
	}
	p := u.Path
	if runtime.GOOS == "windows" && len(p) >= 3 && p[0] == '/' && p[2] == ':' {
		p = p[1:]
	}
	return filepath.FromSlash(p)
}
