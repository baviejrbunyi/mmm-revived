package storage

import (
	"database/sql"
	"net/url"
	"path/filepath"
	"strings"

	engine "modernc.org/sqlite"
)

func openSQLite(path string, pragmas ...string) (*sql.DB, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	name := filepath.ToSlash(absolute)
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}
	parameters := url.Values{"_pragma": pragmas, "_txlock": {"immediate"}}
	uri := url.URL{Scheme: "file", Path: name, RawQuery: parameters.Encode()}
	connector, err := engine.NewConnector(uri.String())
	if err != nil {
		return nil, err
	}
	return sql.OpenDB(connector), nil
}
