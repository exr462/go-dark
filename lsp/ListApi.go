package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
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
		var contentLength int

		// 1. Read headers until we hit the blank line (\r\n) separating headers from payload
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return // Stream closed or process exited
			}

			// Clean line endings for uniform parsing
			line = strings.TrimRight(line, "\r\n")

			// Blank line means we reached the end of headers
			if line == "" {
				break
			}

			// Extract the Content-Length value
			if after, ok := strings.CutPrefix(line, "Content-Length:"); ok {
				sizeStr := strings.TrimSpace(after)
				if size, err := strconv.Atoi(sizeStr); err == nil {
					contentLength = size
				}
			}
		}

		if contentLength <= 0 {
			continue // Safeguard against malformed or missing headers
		}

		// 2. Read exactly the specified number of bytes for the JSON payload
		payload := make([]byte, contentLength)
		_, err := io.ReadFull(reader, payload)
		if err != nil {
			return
		}

		// 3. Dispatch raw payload safely to the Bubble Tea channel listener
		incoming <- payload
	}
}

func (c *Client) Close() {
	c.stdin.Close()
	c.stdout.Close()
	_ = c.cmd.Process.Kill()
}
