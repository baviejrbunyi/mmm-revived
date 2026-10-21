package account

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mmm-revived/internal/database"
	"mmm-revived/internal/json"
	"mmm-revived/internal/model"
)

const legacyAccountsMigration = "legacy-accounts-v1"

type recordKey struct {
	category model.RecordCategory
	id       string
}

func intRecordKey[K ~int32 | ~int64](key K) string { return strconv.FormatInt(int64(key), 10) }

func textRecordKey[K ~string](key K) string { return string(key) }

func recordsFor[K comparable, V any](out map[recordKey][]byte, category model.RecordCategory, values map[K]V, format func(K) string) error {
	for key, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		out[recordKey{category, format(key)}] = data
	}
	return nil
}

func accountRecords(a *Account) (map[recordKey][]byte, error) {
	out := make(map[recordKey][]byte, len(a.Draw.History)+len(a.Dispatch.Missions)+len(a.ShopPurchases)+len(a.Items)+len(a.Heroes)+len(a.Artifacts)+len(a.Highlights)+len(a.Guides)+len(a.Missions)+len(a.Passed)+len(a.Tasks)+len(a.Chapters)+len(a.Activities))
	for _, err := range []error{
		recordsFor(out, model.RecordDrawHistory, a.Draw.History, intRecordKey), recordsFor(out, model.RecordDispatch, a.Dispatch.Missions, textRecordKey),
		recordsFor(out, model.RecordShop, a.ShopPurchases, intRecordKey), recordsFor(out, model.RecordItem, a.Items, intRecordKey),
		recordsFor(out, model.RecordHero, a.Heroes, intRecordKey), recordsFor(out, model.RecordArtifact, a.Artifacts, textRecordKey),
		recordsFor(out, model.RecordHighlight, a.Highlights, textRecordKey), recordsFor(out, model.RecordGuide, a.Guides, textRecordKey),
		recordsFor(out, model.RecordMission, a.Missions, intRecordKey), recordsFor(out, model.RecordPassed, a.Passed, intRecordKey),
		recordsFor(out, model.RecordTask, a.Tasks, intRecordKey), recordsFor(out, model.RecordChapter, a.Chapters, intRecordKey),
		recordsFor(out, model.RecordActivity, a.Activities, intRecordKey),
	} {
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func profileData(a *Account) ([]byte, error) {
	profile := *a
	profile.Draw.History = nil
	profile.Dispatch.Missions = nil
	profile.ShopPurchases = nil
	profile.Items, profile.Heroes, profile.Artifacts, profile.Highlights = nil, nil, nil, nil
	profile.Guides, profile.Missions, profile.Passed, profile.Tasks = nil, nil, nil, nil
	profile.Chapters, profile.Activities = nil, nil
	return json.Marshal(profile)
}

func (s *Store) initializeDatabase() error {
	ctx := context.Background()
	_, err := s.Database.Queries.GetMigration(ctx, legacyAccountsMigration)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	paths, err := filepath.Glob(filepath.Join(s.Dir, "*.json"))
	if err != nil {
		return err
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var a Account
		if err = json.Unmarshal(data, &a); err != nil {
			return fmt.Errorf("migrate %s: %w", filepath.Base(path), err)
		}
		if a.UID == 0 || a.Name == "" {
			return fmt.Errorf("invalid legacy account %s", filepath.Base(path))
		}
		count, err := s.Database.Queries.AccountExists(ctx, a.UID)
		if err != nil {
			return err
		}
		if count != 0 {
			continue
		}
		a.Initialize(s.Catalog, s.Rules)
		if err = s.save(&a); err != nil {
			return err
		}
	}
	return s.Database.Queries.SaveMigration(ctx, legacyAccountsMigration)
}

func (s *Store) save(a *Account) error { return s.saveAtomic(a, nil) }

func (s *Store) Update(a *Account, operation func(*sql.Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a.RegenerateEnergy(s.Rules, time.Now().Unix())
	return s.saveAtomic(a, operation)
}

func (s *Store) saveAtomic(a *Account, operation func(*sql.Tx) error) error {
	var err error
	if s.saved[a.UID] == nil {
		if s.saved[a.UID], err = s.readRecords(a.UID); err != nil {
			return err
		}
	}
	var original []byte
	if operation != nil {
		if original, err = json.Marshal(a); err != nil {
			return err
		}
	}
	tx, err := s.Database.SQL.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		tx.Rollback()
		if !committed && original != nil {
			var restored Account
			if json.Unmarshal(original, &restored) == nil {
				*a = restored
			}
		}
	}()
	if s.Rules != nil {
		a.InitializeJourney(s.Rules, time.Now().Unix())
	}
	if operation != nil {
		if err = operation(tx); err != nil {
			return err
		}
	}
	records, err := s.write(a, tx)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	committed = true
	s.saved[a.UID] = records
	return nil
}

func (s *Store) write(a *Account, tx *sql.Tx) (map[recordKey][]byte, error) {
	if s.Rules != nil {
		if err := s.refreshJourneyRelationalTasks(tx, a); err != nil {
			return nil, err
		}
		if err := a.ProcessAutomaticTasks(s.Rules, time.Now().Unix()); err != nil {
			return nil, err
		}
	}
	profile, err := profileData(a)
	if err != nil {
		return nil, err
	}
	records, err := accountRecords(a)
	if err != nil {
		return nil, err
	}
	previous := s.saved[a.UID]
	if previous == nil {
		if previous, err = s.readRecords(a.UID); err != nil {
			return nil, err
		}
	}
	q, ctx := s.Database.Queries.WithTx(tx), context.Background()
	if err = q.UpsertAccount(ctx, database.UpsertAccountParams{UID: a.UID, Login: a.Name, DisplayName: a.TeamName, SearchName: strings.ToLower(a.TeamName), Level: int64(a.Level), LastLogin: a.LastLogin, Profile: profile}); err != nil {
		return nil, err
	}
	for key, payload := range records {
		if bytes.Equal(previous[key], payload) {
			continue
		}
		if err = q.UpsertRecord(ctx, database.UpsertRecordParams{UID: a.UID, Category: key.category, ID: key.id, Payload: payload}); err != nil {
			return nil, err
		}
	}
	for key := range previous {
		if _, ok := records[key]; ok {
			continue
		}
		if err = q.DeleteRecord(ctx, database.DeleteRecordParams{UID: a.UID, Category: key.category, ID: key.id}); err != nil {
			return nil, err
		}
	}
	return records, nil
}

func (s *Store) readRecords(uid model.UserID) (map[recordKey][]byte, error) {
	rows, err := s.Database.Queries.ListRecords(context.Background(), uid)
	if err != nil {
		return nil, err
	}
	out := make(map[recordKey][]byte, len(rows))
	for _, row := range rows {
		out[recordKey{model.RecordCategory(row.Category), row.ID}] = row.Payload
	}
	return out, nil
}

func decodeRecord[K comparable, V any](values map[K]V, key K, payload []byte) error {
	var value V
	if err := json.Unmarshal(payload, &value); err != nil {
		return err
	}
	values[key] = value
	return nil
}

func (s *Store) restore(uid model.UserID, profile []byte) (*Account, error) {
	var a Account
	if err := json.Unmarshal(profile, &a); err != nil {
		return nil, err
	}
	a.Draw.History = map[int64]model.DrawRecord{}
	a.Dispatch.Missions = map[model.DispatchID]model.DispatchMission{}
	a.ShopPurchases = map[model.ShopID]model.ShopPurchase{}
	a.Items = map[model.ItemID]int64{}
	a.Heroes = map[model.HeroID]model.Hero{}
	a.Artifacts = map[model.InstanceID]model.Artifact{}
	a.Highlights = map[model.InstanceID]model.Highlight{}
	a.Guides = map[string]string{}
	a.Missions = map[model.MissionID]model.Mission{}
	a.Passed = map[model.MissionID]model.MissionProgress{}
	a.Tasks = map[model.TaskID]model.Task{}
	a.Chapters = map[model.ChapterID]model.ChapterProgress{}
	a.Activities = map[model.ActivityID]model.PVPActivity{}
	records, err := s.readRecords(uid)
	if err != nil {
		return nil, err
	}
	for key, payload := range records {
		n, numberError := strconv.ParseInt(key.id, 10, 64)
		switch key.category {
		case model.RecordDrawHistory:
			err = decodeRecord(a.Draw.History, n, payload)
		case model.RecordDispatch:
			err = decodeRecord(a.Dispatch.Missions, model.DispatchID(key.id), payload)
		case model.RecordShop:
			err = decodeRecord(a.ShopPurchases, model.ShopID(n), payload)
		case model.RecordItem:
			err = decodeRecord(a.Items, model.ItemID(n), payload)
		case model.RecordHero:
			err = decodeRecord(a.Heroes, model.HeroID(n), payload)
		case model.RecordArtifact:
			err = decodeRecord(a.Artifacts, model.InstanceID(key.id), payload)
		case model.RecordHighlight:
			err = decodeRecord(a.Highlights, model.InstanceID(key.id), payload)
		case model.RecordGuide:
			err = decodeRecord(a.Guides, key.id, payload)
		case model.RecordMission:
			err = decodeRecord(a.Missions, model.MissionID(n), payload)
		case model.RecordPassed:
			err = decodeRecord(a.Passed, model.MissionID(n), payload)
		case model.RecordTask:
			err = decodeRecord(a.Tasks, model.TaskID(n), payload)
		case model.RecordChapter:
			err = decodeRecord(a.Chapters, model.ChapterID(n), payload)
		case model.RecordActivity:
			err = decodeRecord(a.Activities, model.ActivityID(n), payload)
		default:
			return nil, fmt.Errorf("unknown account record category %s", key.category)
		}
		if key.category != model.RecordDispatch && key.category != model.RecordArtifact && key.category != model.RecordHighlight && key.category != model.RecordGuide && numberError != nil {
			return nil, numberError
		}
		if err != nil {
			return nil, err
		}
	}
	s.saved[uid] = records
	return &a, nil
}
