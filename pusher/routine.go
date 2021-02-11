package pusher

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/Masterminds/semver/v3"
	"github.com/wahabmk/helm-pusher/pkg/helm"
	"github.com/wahabmk/helm-pusher/pkg/rand"
	"helm.sh/helm/v3/pkg/chart"
)

var (
	httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
)

// Routine has all the fields that each go-routine needs.
type Routine struct {
	N              int64
	Errors         int64
	Chart          *chart.Chart
	RepeatFailures bool
}

// Do creates and pushes `N` charts with each chart having random number of versions upto `versions`.
func (r *Routine) Do(versions int64) {
	for r.N > 0 {
		name, err := r.generateName()
		if err != nil {
			continue
		}

		_versions := r.versionsToCreate(versions)
		r.N -= _versions

		for i := _versions; i > 0; i-- {
			version, err := r.generateVersion(_versions)
			if err != nil {
				continue
			}

			reader, err := r.generateChart(name, version)
			if err != nil {
				continue
			}

			if err = r.pushChart(reader, false); err != nil {
				continue
			}
		}
	}
}

func (r *Routine) generateName() (string, error) {
	name, err := rand.String(10)
	if err != nil {
		return "", fmt.Errorf("error generating rand string for name: %w", err)
	}

	return name, nil
}

func (r *Routine) generateVersion(versions int64) (string, error) {
	var (
		err        error
		prerelease string
		version    *semver.Version
	)

	defer func() {
		if err != nil {
			if r.RepeatFailures {
				r.N++
			} else {
				r.Errors++
			}
		}
	}()

	major := rand.Int64(0, versions)
	minor := rand.Int64(0, versions)
	patch := rand.Int64(0, versions)
	prerelease, err = rand.String(rand.Int64(versions, versions*2))
	if err != nil {
		return "", fmt.Errorf("error generating rand string for prerelease: %w", err)
	}

	ver := fmt.Sprintf("%d.%d.%d-%s", major, minor, patch, prerelease)
	version, err = semver.NewVersion(ver)
	if err != nil {
		return "", err
	}

	return version.String(), nil
}

func (r *Routine) versionsToCreate(versions int64) int64 {
	var _versions int64

	if versions <= 0 {
		_versions = 0
	} else if versions == 1 {
		_versions = 1
	} else if versions > r.N {
		_versions = r.N
	} else {
		_versions = rand.Int64(0, versions)
	}

	return _versions
}

func (r *Routine) generateChart(name, version string) (io.Reader, error) {
	r.Chart.Metadata.Name = name
	r.Chart.Metadata.Version = version

	buf, err := helm.PackageChart(r.Chart)
	if err != nil {
		if r.RepeatFailures {
			r.N++
		} else {
			r.Errors++
		}
		return nil, fmt.Errorf("failed to package chart: %w", err)
	}

	return buf, nil
}

func (r *Routine) pushChart(reader io.Reader, force bool) error {
	return r.pushContent(username, password, reader, "application/octet-stream", &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   fmt.Sprintf("/charts/api/%s/%s/charts", namespace, repository),
	}, force)
}

func (r *Routine) pushContent(username, password string, reader io.Reader, contentType string, u *url.URL, force bool) error {
	var (
		err  error
		b    []byte
		req  *http.Request
		resp *http.Response
	)

	defer func() {
		if err != nil {
			if r.RepeatFailures {
				r.N++
			} else {
				r.Errors++
			}
		}
	}()

	b, err = ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	req, err = http.NewRequest(http.MethodPost, u.String(), bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", contentType)

	if force {
		q := req.URL.Query()
		q.Add("force", "")
		req.URL.RawQuery = q.Encode()
	}

	resp, err = httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		err = fmt.Errorf("returned with status %d", resp.StatusCode)
		return err
	}

	return nil
}
