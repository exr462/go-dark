package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// Request represents a generic JSON-RPC 2.0 request
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// Response represents a generic JSON-RPC 2.0 response
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   interface{}     `json:"error,omitempty"`
}

// Client wraps our background OS process and IO streams
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	idSeq  int
	mu     sync.Mutex
}

func NewClient(binary string, args []string) (*Client, error) {
	cmd := exec.Command(binary, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &Client{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
	}, nil
}

// Call sends a synchronous-style JSON-RPC request over standard protocol mapping
func (c *Client) Call(method string, params interface{}) (int, error) {
	c.mu.Lock()
	c.idSeq++
	id := c.idSeq
	c.mu.Unlock()

	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}

	// LSP requires a specific Content-Length HTTP-like header format over stdin
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(payload))

	if _, err := c.stdin.Write([]byte(header)); err != nil {
		return 0, err
	}
	if _, err := c.stdin.Write(payload); err != nil {
		return 0, err
	}

	return id, nil
}

// Listen reads continuous LSP notifications and responses on a background loop
func (c *Client) Listen(incoming chan<- []byte) {
	reader := bufio.NewReader(c.stdout)
	for {
		// Real LSP engines send Content-Length headers first.
		// For brevity, your parser looks for the double newline before payload blocks.
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		if line == "\r\n" {
			// This signals the start of the JSON payload.
			// In production, parse Content-Length to read exactly X bytes.
			var jsonBytes []byte
			// Basic reader fallback illustration:
			jsonBytes, _, err = reader.ReadLine()
			if err == nil {
				incoming <- jsonBytes
			}
		}
	}
}

func (c *Client) Close() {
	c.stdin.Close()
	c.stdout.Close()
	_ = c.cmd.Process.Kill()
}
