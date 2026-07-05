package backend

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const DefaultDownloaderUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"

func NewRequestWithDefaultHeaders(method string, rawURL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, rawURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", DefaultDownloaderUserAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")

	return req, nil
}

// SimpleHealthCheck performs a quick HEAD request to verify an API endpoint is reachable.
func SimpleHealthCheck(apiURL string) (bool, error) {
	apiURL = strings.TrimRight(strings.TrimSpace(apiURL), "/")
	if apiURL == "" {
		return false, fmt.Errorf("URL is empty")
	}

	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest("HEAD", apiURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", DefaultDownloaderUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 500, nil
}
