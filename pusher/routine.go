package pusher

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"

	"github.com/Masterminds/semver/v3"
	"github.com/wahabmk/helm-pusher/pkg/helm"
	"github.com/wahabmk/helm-pusher/pkg/random"
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

// routine has all the fields that each go-routine needs.
type routine struct {
	id             int64
	nCharts        int64
	errors         int64
	chart          *chart.Chart
	repeatFailures bool
	errorKinds     map[string]interface{}
}

// Push creates and pushes `N` charts with each chart having random number of versions upto `versions`.
//
// TODO: Sometimes pushing returns with status 422. Status 422 in MSR is returned when the chart already exists.
// Maybe there is some issue with randomness that creates charts with equal name?
// This bug has been observed fairly consistently with the following parameters:
// 	nCharts        = 10
//	nVersions      = 2
//	nRoutines      = 2
func (r *routine) push(versions int64, url, username, password string) {
	entropy := random.New(r.id + 1)

	for r.nCharts > 0 {
		name, err := r.generateName(entropy)
		if err != nil {
			continue
		}

		_versions := r.versionsToCreate(versions)
		r.nCharts -= _versions

		for i := _versions; i > 0; i-- {
			version, err := r.generateVersion(entropy, _versions)
			if err != nil {
				continue
			}

			reader, err := r.generateChart(name, version)
			if err != nil {
				continue
			}

			if err = r.pushChart(reader, username, password, url, false); err != nil {
				continue
			}
		}
	}
}

func (r *routine) generateName(entropy *random.Entropy) (string, error) {
	u, err := entropy.String()
	if err != nil {
		r.errors++
		r.errorKinds[err.Error()] = nil
		if r.repeatFailures {
			r.nCharts++
		}
		return "", fmt.Errorf("error generating name: %w", err)
	}

	return u, nil
}

func (r *routine) generateVersion(entropy *random.Entropy, versions int64) (string, error) {
	var (
		err        error
		prerelease string
		version    *semver.Version
	)

	defer func() {
		if err != nil {
			r.errors++
			r.errorKinds[err.Error()] = nil
			if r.repeatFailures {
				r.nCharts++
			}
		}
	}()

	major := rand.Int63()
	minor := rand.Int63()
	patch := rand.Int63()
	prerelease, err = entropy.String()
	if err != nil {
		err = fmt.Errorf("error generating prerelease: %w", err)
		return "", err
	}

	ver := fmt.Sprintf("%d.%d.%d-%s", major, minor, patch, prerelease)
	version, err = semver.NewVersion(ver)
	if err != nil {
		return "", err
	}

	return version.String(), nil
}

func (r *routine) versionsToCreate(versions int64) int64 {
	var _versions int64

	if versions <= 0 {
		_versions = 0
	} else if versions == 1 {
		_versions = 1
	} else if versions > r.nCharts {
		_versions = r.nCharts
	} else {
		_versions = random.Int63n(1, versions+1)
	}

	return _versions
}

func (r *routine) generateChart(name, version string) (io.Reader, error) {
	r.chart.Metadata.Name = name
	r.chart.Metadata.Version = version

	buf, err := helm.PackageChart(r.chart)
	if err != nil {
		r.errors++
		r.errorKinds[err.Error()] = nil
		if r.repeatFailures {
			r.nCharts++
		}

		return nil, fmt.Errorf("failed to package chart: %w", err)
	}

	return buf, nil
}

func (r *routine) pushChart(reader io.Reader, username, password, u string, force bool) error {
	_u, err := url.Parse(u)
	if err != nil {
		return err
	}

	return r.pushContent(username, password, reader, "application/octet-stream", _u, force)
}

func (r *routine) pushToArtifactory(username, password string, reader io.Reader, u *url.URL) error {
	var (
		err  error
		b    []byte
		req  *http.Request
		resp *http.Response
	)

	defer func() {
		if err != nil {
			r.errors++
			r.errorKinds[err.Error()] = nil
			if r.repeatFailures {
				r.nCharts++
			}
		}
	}()

	b, err = ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	req, err = http.NewRequest(http.MethodPut, u.String(), bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	if username != "" {
		req.SetBasicAuth(username, password)
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

func (r *routine) pushContent(username, password string, reader io.Reader, contentType string, u *url.URL, force bool) error {
	var (
		err  error
		b    []byte
		req  *http.Request
		resp *http.Response
	)

	defer func() {
		if err != nil {
			r.errors++
			r.errorKinds[err.Error()] = nil
			if r.repeatFailures {
				r.nCharts++
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

	if username != "" {
		req.SetBasicAuth(username, password)
	}
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
