package account

import (
	"context"
	"errors"
	"strconv"

	"mmm-revived/internal/json"
	"mmm-revived/internal/model"
)

type PublicProfile struct {
	Guild     model.GuildID
	GuildRole model.GuildRole
	Account   Account
	Heroes    int64
	Online    int64
	Offline   int64
}

func encodeProfileIDs(ids []model.UserID) (string, error) {
	if len(ids) == 0 || len(ids) > 100 {
		return "", errors.New("invalid public profile batch size")
	}
	encoded, err := json.Marshal(ids)
	return string(encoded), err
}

func (s *Store) PublicProfiles(ids []model.UserID, includeHeroes bool) (map[model.UserID]PublicProfile, error) {
	encoded, err := encodeProfileIDs(ids)
	if err != nil {
		return nil, err
	}
	return s.publicProfiles(context.Background(), encoded, includeHeroes)
}

func (s *Store) publicProfiles(ctx context.Context, encoded string, includeHeroes bool) (map[model.UserID]PublicProfile, error) {
	rows, err := s.Database.Queries.GetPublicProfiles(ctx, encoded)
	if err != nil {
		return nil, err
	}
	result := make(map[model.UserID]PublicProfile, len(rows))
	for _, row := range rows {
		var a Account
		if err = json.Unmarshal(row.Profile, &a); err != nil {
			return nil, err
		}
		result[row.UID] = PublicProfile{Account: a, Heroes: row.HeroCount, Online: row.Online, Offline: row.Offline, Guild: model.GuildID(row.GuildID), GuildRole: model.GuildRole(row.GuildRole)}
	}
	passed, err := s.Database.Queries.GetPublicPassed(ctx, encoded)
	if err != nil {
		return nil, err
	}
	for _, row := range passed {
		profile, ok := result[row.UID]
		if !ok {
			return nil, errors.New("passed mission references missing public profile")
		}
		id, err := strconv.ParseInt(row.ID, 10, 64)
		if err != nil {
			return nil, err
		}
		if profile.Account.Passed == nil {
			profile.Account.Passed = map[model.MissionID]model.MissionProgress{}
		}
		if err = decodeRecord(profile.Account.Passed, model.MissionID(id), row.Payload); err != nil {
			return nil, err
		}
		result[row.UID] = profile
	}
	if !includeHeroes {
		return result, nil
	}
	heroes, err := s.Database.Queries.GetPublicHeroes(ctx, encoded)
	if err != nil {
		return nil, err
	}
	for _, row := range heroes {
		profile, ok := result[row.UID]
		if !ok {
			return nil, errors.New("hero references missing public profile")
		}
		if profile.Account.Heroes == nil {
			profile.Account.Heroes = map[model.HeroID]model.Hero{}
		}
		var hero model.Hero
		if err = json.Unmarshal(row.Payload, &hero); err != nil {
			return nil, err
		}
		profile.Account.Heroes[hero.ID] = hero
		result[row.UID] = profile
	}
	return result, nil
}
