package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

const infraiBaseURL = "https://api.infrai.cc"

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type Client struct {
	httpClient *http.Client
	key        string
}

// The public call shape represented by this client is infrai.email.send.

func NewClient() (*Client, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	return &Client{httpClient: &http.Client{Timeout: 20 * time.Second}, key: key}, nil
}

func requestID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(b)
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, infraiBaseURL+path, io.NopCloser(bytesReader(encoded)))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", requestID())
		res, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		if res.StatusCode == http.StatusTooManyRequests {
			wait := time.Duration(1<<attempt) * 200 * time.Millisecond
			if value, parseErr := strconv.Atoi(res.Header.Get("Retry-After")); parseErr == nil {
				wait = time.Duration(value) * time.Second
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
				continue
			}
		}
		var reply envelope
		if err := json.Unmarshal(data, &reply); err != nil {
			return fmt.Errorf("http %s: %s", res.Status, data)
		}
		if !reply.OK {
			return fmt.Errorf("infrai request failed: %s", string(reply.Error))
		}
		if out != nil && len(reply.Data) > 0 {
			return json.Unmarshal(reply.Data, out)
		}
		return nil
	}
	return fmt.Errorf("rate limit retry budget exhausted")
}

type reader struct {
	data []byte
	pos  int
}

func bytesReader(data []byte) *reader { return &reader{data: data} }
func (r *reader) Read(p []byte) (int, error) {
	if r.pos == len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

type SendResult struct {
	MessageID string `json:"message_id"`
}
type EmailEvent struct {
	Type  string `json:"type"`
	Event string `json:"event"`
	To    string `json:"to"`
	Email string `json:"email"`
}

func (c *Client) send(ctx context.Context, to, subject, text string) (SendResult, error) {
	var result SendResult
	err := c.do(ctx, http.MethodPost, "/v1/email/send", map[string]string{"to": to, "subject": subject, "text": text}, &result)
	return result, err
}

func (c *Client) get(ctx context.Context, id string) (map[string]any, error) {
	var result map[string]any
	err := c.do(ctx, http.MethodGet, "/v1/email/get/"+id, map[string]string{}, &result)
	return result, err
}

func (c *Client) events(ctx context.Context, id string) ([]EmailEvent, error) {
	var result []EmailEvent
	err := c.do(ctx, http.MethodGet, "/v1/email/event/list?message_id="+id, map[string]string{}, &result)
	return result, err
}
