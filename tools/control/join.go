package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"mmm-revived/internal/client"
)

const joinManifestLimit = 4 << 20

func serverOrigin(value string) (string, error) {
	origin, err := url.Parse(value)
	if err != nil || origin.Scheme != "http" && origin.Scheme != "https" || origin.Hostname() == "" || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" || origin.Path != "" && origin.Path != "/" {
		return "", errors.New("the server address must look like http://<host>:18080")
	}
	origin.Path = ""
	return origin.String(), nil
}

func fetchManifest(ctx context.Context, origin string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, origin+client.PatchListPath, nil)
	if err != nil {
		return nil, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("reach the server at %s: %w", origin, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("the server at %s did not provide a client update (HTTP %d)", origin, response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, joinManifestLimit+1))
	if err != nil {
		return nil, err
	}
	if len(data) > joinManifestLimit {
		return nil, errors.New("the server's update list is too large")
	}
	return data, nil
}

func (w workflow) join(ctx context.Context, directory, server string) error {
	origin, err := serverOrigin(server)
	if err != nil {
		return err
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return err
	}
	if err = client.EnsureClientClosed(absolute); err != nil {
		return err
	}
	if _, err = client.DetectBuild(absolute); err != nil {
		return err
	}
	unlock, err := w.lock()
	if err != nil {
		return err
	}
	defer unlock()
	manifest, err := fetchManifest(ctx, origin)
	if err != nil {
		return err
	}
	staging, err := os.MkdirTemp("", "mmm-revived-join-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	manifestPath := filepath.Join(staging, "patchlist_win32.json")
	if err = os.WriteFile(manifestPath, manifest, 0600); err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	runtimePath, err := client.RuntimeDirectory(home, absolute)
	if err != nil {
		return err
	}
	backup := filepath.Join(w.root, "build", "backups", filepath.Base(runtimePath), time.Now().UTC().Format("20060102T150405.000000000"))
	if err = client.ConfigureNativePatches(client.NativePatchOptions{
		Runtime: runtimePath, ClientDir: absolute, Backup: backup, Debug: false,
		ServerURL: origin, EngineManifest: manifestPath, Content: client.Supported.ContentMarker(),
	}); err != nil {
		if regular(filepath.Join(backup, "native-installation.json")) {
			err = errors.Join(err, client.RestoreNativePatches(backup))
		}
		return err
	}
	fmt.Fprintf(w.output, "Configured %s to play on %s.\nOpen the game and accept its download and restart prompts.\n", absolute, origin)
	fmt.Fprintln(w.output, "Configuration backup:", backup)
	return nil
}
