package helm

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
)

var (
	headerBytes = []byte("+aHR0cHM6Ly95b3V0dS5iZS96OVV6MWljandyTQo=")
)

func PackageChart(c *chart.Chart) (io.Reader, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("chart validation: %w", err)
	}

	// var b []byte
	buf := bytes.NewBuffer([]byte{})

	// Wrap in gzip writer
	zipper := gzip.NewWriter(buf)
	zipper.Header.Extra = headerBytes
	zipper.Header.Comment = "Helm"

	// Wrap in tar writer
	twriter := tar.NewWriter(zipper)
	// rollback := false
	defer func() {
		twriter.Close()
		zipper.Close()
	}()

	if err := writeTarContents(twriter, c, ""); err != nil {
		return nil, err
	}

	return buf, nil
}

func writeTarContents(out *tar.Writer, c *chart.Chart, prefix string) error {
	base := filepath.Join(prefix, c.Name())

	// Pull out the dependencies of a v1 Chart, since there's no way
	// to tell the serializer to skip a field for just this use case
	savedDependencies := c.Metadata.Dependencies
	if c.Metadata.APIVersion == chart.APIVersionV1 {
		c.Metadata.Dependencies = nil
	}
	// Save Chart.yaml
	cdata, err := yaml.Marshal(c.Metadata)
	if c.Metadata.APIVersion == chart.APIVersionV1 {
		c.Metadata.Dependencies = savedDependencies
	}
	if err != nil {
		return err
	}
	if err := writeToTar(out, filepath.Join(base, chartutil.ChartfileName), cdata); err != nil {
		return err
	}

	// Save Chart.lock
	// TODO: remove the APIVersion check when APIVersionV1 is not used anymore
	if c.Metadata.APIVersion == chart.APIVersionV2 {
		if c.Lock != nil {
			ldata, err := yaml.Marshal(c.Lock)
			if err != nil {
				return err
			}
			if err := writeToTar(out, filepath.Join(base, "Chart.lock"), ldata); err != nil {
				return err
			}
		}
	}

	// Save values.yaml
	for _, f := range c.Raw {
		if f.Name == chartutil.ValuesfileName {
			if err := writeToTar(out, filepath.Join(base, chartutil.ValuesfileName), f.Data); err != nil {
				return err
			}
		}
	}

	// Save values.schema.json if it exists
	if c.Schema != nil {
		if !json.Valid(c.Schema) {
			return errors.New("Invalid JSON in " + chartutil.SchemafileName)
		}
		if err := writeToTar(out, filepath.Join(base, chartutil.SchemafileName), c.Schema); err != nil {
			return err
		}
	}

	// Save templates
	for _, f := range c.Templates {
		n := filepath.Join(base, f.Name)
		if err := writeToTar(out, n, f.Data); err != nil {
			return err
		}
	}

	// Save files
	for _, f := range c.Files {
		n := filepath.Join(base, f.Name)
		if err := writeToTar(out, n, f.Data); err != nil {
			return err
		}
	}

	// Save dependencies
	for _, dep := range c.Dependencies() {
		if err := writeTarContents(out, dep, filepath.Join(base, chartutil.ChartsDir)); err != nil {
			return err
		}
	}
	return nil
}

// writeToTar writes a single file to a tar archive.
func writeToTar(out *tar.Writer, name string, body []byte) error {
	// TODO: Do we need to create dummy parent directory names if none exist?
	h := &tar.Header{
		Name:    filepath.ToSlash(name),
		Mode:    0644,
		Size:    int64(len(body)),
		ModTime: time.Now(),
	}
	if err := out.WriteHeader(h); err != nil {
		return err
	}
	_, err := out.Write(body)
	return err
}
