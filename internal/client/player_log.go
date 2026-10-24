package client

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	bootConfigName  = "Marvel MM CN_Data/boot.config"
	bootConfigLimit = 1 << 16
	bootNoLog       = "nolog"
)

type bootSetting struct {
	Key   string
	Value string
}

type bootConfig struct {
	Settings []bootSetting
	Newline  string
}

func parseBootConfig(data []byte) (bootConfig, error) {
	config := bootConfig{Newline: "\n"}
	if bytes.Contains(data, []byte("\r\n")) {
		config.Newline = "\r\n"
	}
	text := strings.TrimSuffix(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if text == "" {
		return config, nil
	}
	seen := map[string]bool{}
	for number, line := range strings.Split(text, "\n") {
		key, value, found := strings.Cut(line, "=")
		if !found || key == "" || strings.TrimSpace(key) != key || strings.ContainsAny(key, "\r\x00") || strings.ContainsAny(value, "\r\x00") {
			return bootConfig{}, fmt.Errorf("invalid boot.config line %d", number+1)
		}
		if seen[key] {
			return bootConfig{}, fmt.Errorf("duplicate boot.config key %q", key)
		}
		seen[key] = true
		config.Settings = append(config.Settings, bootSetting{Key: key, Value: value})
	}
	return config, nil
}

func (c *bootConfig) Remove(key string) bool {
	before := len(c.Settings)
	c.Settings = slices.DeleteFunc(c.Settings, func(setting bootSetting) bool { return setting.Key == key })
	return len(c.Settings) != before
}

func (c bootConfig) Bytes() []byte {
	var buffer bytes.Buffer
	for _, setting := range c.Settings {
		buffer.WriteString(setting.Key)
		buffer.WriteByte('=')
		buffer.WriteString(setting.Value)
		buffer.WriteString(c.Newline)
	}
	return buffer.Bytes()
}

func EnablePlayerLog(clientDir, backupPath string) error {
	root, err := os.OpenRoot(clientDir)
	if err != nil {
		return err
	}
	defer root.Close()
	original, err := readConfigFile(root, bootConfigName, bootConfigLimit)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	config, err := parseBootConfig(original)
	if err != nil {
		return err
	}
	if !config.Remove(bootNoLog) {
		return nil
	}
	updated := config.Bytes()
	if err = os.MkdirAll(backupPath, 0700); err != nil {
		return err
	}
	backup, err := os.OpenRoot(backupPath)
	if err != nil {
		return err
	}
	defer backup.Close()
	if err = writeConfigBackup(backup, filepath.Base(bootConfigName), original); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	token, err := nativeConfigToken()
	if err != nil {
		return err
	}
	return replaceConfigFile(root, bootConfigName, bytes.NewReader(updated), hash(updated), int64(len(updated)), 0644, token)
}
