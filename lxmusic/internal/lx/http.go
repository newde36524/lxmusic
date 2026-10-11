package lx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

const defaultUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// Client is the small HTTP layer shared by every provider. It mirrors the
// desktop app's helper (axios-like `tX`) in the things that matter: a fixed
// timeout, a real User-Agent, redirect following and JSON bodies.
type Client struct {
	HTTP    *http.Client
	UA      string
	Proxy   string // optional HTTP proxy URL
	Verbose bool
}

// NewClient builds a client with sane defaults.
func NewClient() *Client {
	c := &Client{
		HTTP: &http.Client{Timeout: 25 * time.Second},
		UA:   defaultUA,
	}
	if p := os.Getenv("HTTP_PROXY"); p != "" {
		c.Proxy = p
	}
	return c
}

func (c *Client) userAgent() string {
	if c.UA != "" {
		return c.UA
	}
	return defaultUA
}

func (c *Client) log(format string, a ...any) {
	if c.Verbose {
		fmt.Fprintf(os.Stderr, format+"\n", a...)
	}
}

func (c *Client) newRequest(ctx context.Context, method, rawURL string, hdr map[string]string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent())
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	c.log("HTTP %s %s", method, rawURL)
	return req, nil
}

// Response holds a decoded HTTP response.
type Response struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}

// JSON unmarshals the response body into out.
func (r *Response) JSON(out any) error {
	return json.Unmarshal(r.Body, out)
}

// Do performs an arbitrary request.
func (c *Client) Do(ctx context.Context, method, rawURL string, hdr map[string]string, body io.Reader) (*Response, error) {
	req, err := c.newRequest(ctx, method, rawURL, hdr, body)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	c.log("HTTP %s %d (%d bytes)", resp.Request.URL.Host, resp.StatusCode, len(b))
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return &Response{StatusCode: resp.StatusCode, Body: b, Header: resp.Header},
			fmt.Errorf("http %d from %s", resp.StatusCode, resp.Request.URL.Host)
	}
	return &Response{StatusCode: resp.StatusCode, Body: b, Header: resp.Header}, nil
}

// GetJSON issues a GET and decodes the JSON body.
func (c *Client) GetJSON(ctx context.Context, rawURL string, hdr map[string]string, out any) error {
	resp, err := c.Do(ctx, http.MethodGet, rawURL, hdr, nil)
	if err != nil {
		return err
	}
	return resp.JSON(out)
}

// PostJSON issues a POST with a JSON body and decodes the JSON response.
func (c *Client) PostJSON(ctx context.Context, rawURL string, hdr map[string]string, payload, out any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	h := map[string]string{"Content-Type": "application/json"}
	for k, v := range hdr {
		h[k] = v
	}
	resp, err := c.Do(ctx, http.MethodPost, rawURL, h, bytes.NewReader(b))
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	return resp.JSON(out)
}

// PostForm posts form-urlencoded data and decodes the JSON response.
func (c *Client) PostForm(ctx context.Context, rawURL string, hdr map[string]string, form url.Values, out any) error {
	h := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
	for k, v := range hdr {
		h[k] = v
	}
	resp, err := c.Do(ctx, http.MethodPost, rawURL, h, bytes.NewReader([]byte(form.Encode())))
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	return resp.JSON(out)
}

// DownloadFile streams a URL to a local file, returning the number of bytes.
func (c *Client) DownloadFile(ctx context.Context, rawURL, path string, hdr map[string]string) (int64, error) {
	return c.DownloadFileOn(ctx, rawURL, path, hdr, nil)
}

// DownloadFileOn is DownloadFile with a progress callback. on is called after
// each chunk with the bytes written so far and the total the server announced;
// total is 0 when there is no Content-Length, in which case the caller should
// show a plain byte count rather than a percentage. A nil callback is the
// ordinary download, and the body is not wrapped at all.
func (c *Client) DownloadFileOn(ctx context.Context, rawURL, path string, hdr map[string]string,
	on func(done, total int64)) (int64, error) {
	// c.HTTP.Timeout also covers the body, so the 25s API deadline would kill a
	// multi-hundred-MB lossless download on a slow link before the caller's
	// context ever did. The context is the deadline here.
	httpc := *c.HTTP
	httpc.Timeout = 0

	req, err := c.newRequest(ctx, http.MethodGet, rawURL, hdr, nil)
	if err != nil {
		return 0, err
	}
	resp, err := httpc.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("http %d downloading %s", resp.StatusCode, resp.Request.URL.Host)
	}
	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	src := io.Reader(resp.Body)
	if on != nil {
		src = &reportingReader{r: resp.Body, on: on, total: resp.ContentLength}
	}
	n, err := io.Copy(f, src)
	c.log("saved %s (%d bytes)", path, n)
	return n, err
}

// reportingReader reports running byte counts as the body is read. total is the
// ContentLength the server announced, which is 0 on chunked responses and on
// some redirect hops that drop the header.
type reportingReader struct {
	r     io.Reader
	on    func(done, total int64)
	total int64
	done  int64
}

func (p *reportingReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.done += int64(n)
		p.on(p.done, p.total)
	}
	return n, err
}

// GetBytes fetches raw bytes from a URL.
func (c *Client) GetBytes(ctx context.Context, rawURL string, hdr map[string]string) ([]byte, error) {
	resp, err := c.Do(ctx, http.MethodGet, rawURL, hdr, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
