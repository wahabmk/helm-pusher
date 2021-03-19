package pusher

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

var (
	httpClient = &http.Client{
		Transport: &http.Transport{
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 30 * time.Second,
	}
)

var ErrConflict = errors.New("chart conflicts with an existing chart version")

func pushChart(ctx context.Context, reader io.Reader, u string, force bool) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, reader)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	if force {
		q := req.URL.Query()
		q.Add("force", "")
		req.URL.RawQuery = q.Encode()
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Consume the response so the connection can be reused.
	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnprocessableEntity {
		return ErrConflict
	} else if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("returned with status %s\n%s", resp.Status, string(body))
	}
	return nil
}
