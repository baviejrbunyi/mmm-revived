package account

import (
	"context"

	"mmm-revived/internal/model"
)

func (s *Store) NetworkProfiles(ids []model.UserID) (map[model.UserID]PublicProfile, error) {
	if len(ids) == 0 {
		return map[model.UserID]PublicProfile{}, nil
	}
	encoded, err := encodeProfileIDs(ids)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	profiles, err := s.publicProfiles(ctx, encoded, true)
	if err != nil {
		return nil, err
	}
	rows, err := s.Database.Queries.GetPublicEquipment(ctx, encoded)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		profile := profiles[row.UID]
		if profile.Account.Artifacts == nil {
			profile.Account.Artifacts = map[model.InstanceID]model.Artifact{}
		}
		if profile.Account.Highlights == nil {
			profile.Account.Highlights = map[model.InstanceID]model.Highlight{}
		}
		switch row.Category {
		case model.RecordArtifact:
			if err = decodeRecord(profile.Account.Artifacts, model.InstanceID(row.ID), row.Payload); err != nil {
				return nil, err
			}
		case model.RecordHighlight:
			if err = decodeRecord(profile.Account.Highlights, model.InstanceID(row.ID), row.Payload); err != nil {
				return nil, err
			}
		}
		profiles[row.UID] = profile
	}
	return profiles, nil
}
