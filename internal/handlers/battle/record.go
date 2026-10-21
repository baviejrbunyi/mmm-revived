package battle

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"io"

	"mmm-revived/internal/account"
	"mmm-revived/internal/model"
	"mmm-revived/internal/wire"
)

func RecordDeployment(value any) (map[model.HeroID]model.DeploymentHero, error) {
	groups, okay := value.([]any)
	if !okay || len(groups) == 0 {
		return nil, fmt.Errorf("%w: asynchronous deployment groups are missing", account.ErrProgression)
	}
	rows, okay := groups[0].([]any)
	if !okay || len(rows) == 0 || len(rows) > 3 {
		return nil, fmt.Errorf("%w: invalid asynchronous deployment size", account.ErrProgression)
	}
	heroes := make(map[model.HeroID]model.DeploymentHero, len(rows))
	for _, item := range rows {
		row, okay := item.([]any)
		if !okay || len(row) != 4 {
			return nil, fmt.Errorf("%w: invalid asynchronous deployment row", account.ErrProgression)
		}
		slot, slotOK := Integer(row[0])
		hero, heroOK := Integer(row[1])
		order, orderOK := Integer(row[3])
		var trial bool
		if value, boolean := row[2].(bool); boolean {
			trial = !value
		} else {
			value, integer := Integer(row[2])
			trial = integer && value == 0
		}
		if !slotOK || !heroOK || !orderOK || hero < 1 || hero > 1<<31-1 || slot < -1<<31 || slot > 1<<31-1 || order < 1 || order > 1000000 || !trial {
			return nil, fmt.Errorf("%w: invalid asynchronous deployment values", account.ErrProgression)
		}
		id := model.HeroID(hero)
		if _, duplicate := heroes[id]; duplicate {
			return nil, fmt.Errorf("%w: duplicate asynchronous hero", account.ErrProgression)
		}
		heroes[id] = model.DeploymentHero{Slot: int(slot), Hero: id, Order: int(order)}
	}
	return heroes, nil
}

func DecodeRecord(record string) (map[any]any, error) {
	if len(record) == 0 || len(record) > 1<<20 {
		return nil, fmt.Errorf("%w: invalid asynchronous battle record size", account.ErrProgression)
	}
	compressed, err := base64.StdEncoding.DecodeString(record)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid battle record encoding", account.ErrProgression)
	}
	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid battle record compression", account.ErrProgression)
	}
	defer r.Close()
	data, err := io.ReadAll(io.LimitReader(r, 4<<20+1))
	if err != nil || len(data) > 4<<20 {
		return nil, fmt.Errorf("%w: battle record expansion exceeded", account.ErrProgression)
	}
	value, err := wire.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid battle record", account.ErrProgression)
	}
	result, ok := value.(map[any]any)
	if !ok {
		return nil, fmt.Errorf("%w: battle record must be a dictionary", account.ErrProgression)
	}
	return result, nil
}

func Integer(value any) (int64, bool) {
	switch n := value.(type) {
	case int64:
		return n, true
	case uint64:
		if n <= 1<<63-1 {
			return int64(n), true
		}
	case int:
		return int64(n), true
	}
	return 0, false
}

func Dictionary(value any) map[any]any {
	result, _ := value.(map[any]any)
	return result
}
