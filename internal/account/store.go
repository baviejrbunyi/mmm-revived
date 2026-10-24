package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mmm-revived/internal/json"
	"mmm-revived/internal/model"
	"mmm-revived/internal/social"
	"mmm-revived/internal/storage"
)

type Store struct {
	Social   *social.Repository
	Dir      string
	Catalog  Catalog
	Rules    *Rules
	Database *storage.Database
	Start    Start
	Fresh    bool
	mu       sync.Mutex
	saved    map[model.UserID]map[recordKey][]byte
	mailMu   sync.Mutex
	mailed   map[model.UserID]struct{}
	bossDone map[string]bool
}

func Open(dir, catalogPath string) (*Store, error) {
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		return nil, err
	}
	s := &Store{Dir: dir, saved: map[model.UserID]map[recordKey][]byte{}}
	if err = json.Unmarshal(data, &s.Catalog); err != nil {
		return nil, err
	}
	if len(s.Catalog.Heroes) == 0 || len(s.Catalog.Items) == 0 {
		return nil, errors.New("catalog is empty")
	}
	s.Rules, err = LoadRules(filepath.Join(filepath.Dir(catalogPath), "rules.json"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load rules: %w", err)
	}
	s.Catalog.Cosmetics = map[int][]model.ItemID{}
	s.Catalog.SkinItems = map[model.ItemID]model.SkinID{}
	if s.Rules != nil {
		for _, id := range s.Catalog.Items {
			kind := s.Rules.ItemTypes[id]
			s.Catalog.Cosmetics[kind] = append(s.Catalog.Cosmetics[kind], id)
			if rule := s.Rules.ItemRules[id]; rule.Type == 1032 && rule.Parameter != 0 {
				s.Catalog.SkinItems[id] = model.SkinID(rule.Parameter)
			}
		}
		s.Catalog.DefaultSkins = s.Rules.Defaults.Retail.DefaultSkins
	}
	if err = os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	s.Database, err = storage.Open(filepath.Join(dir, "mmm-revived.db"), storage.DefaultOptions())
	if err != nil {
		return nil, err
	}
	if err = s.prepare(); err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) prepare() error {
	if err := s.initializeDatabase(); err != nil {
		return err
	}
	if err := s.migrateNativeUIDs(); err != nil {
		return err
	}
	var err error
	if s.Social, err = social.Open(s.Database); err != nil {
		return err
	}
	if err = s.RecoverNetworkBattles(); err != nil {
		return fmt.Errorf("recover interrupted network battles: %w", err)
	}
	return nil
}

func (s *Store) Close() error { return s.Database.Close() }

func (s *Store) Login(name string) (*Account, error) {
	name = strings.TrimSpace(name)
	if len(name) == 0 || len(name) > 64 || strings.ContainsAny(name, "\r\n\x00") {
		return nil, errors.New("invalid account name")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	row, err := s.Database.Queries.GetAccount(context.Background(), name)
	if errors.Is(err, sql.ErrNoRows) {
		return s.createAccount(name)
	}
	if err != nil {
		return nil, err
	}
	a, err := s.restore(row.UID, row.Profile)
	if err != nil {
		return nil, err
	}
	a.LastLogin = time.Now().Unix()
	a.RegenerateEnergy(s.Rules, a.LastLogin)
	a.Initialize(s.Catalog, s.Rules)
	a.ExpireCosmetics(s.Catalog, a.LastLogin)
	if err = a.PrepareDispatch(s.Rules, a.LastLogin); err != nil {
		return nil, err
	}
	if err = s.save(a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Store) Save(a *Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save(a)
}
