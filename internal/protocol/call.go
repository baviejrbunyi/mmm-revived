package protocol

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"mmm-revived/internal/json"
	"mmm-revived/internal/model"
)

type Call struct {
	Name string
	Args []any
}

func Integer(v any) (int64, error) {
	switch n := v.(type) {
	case int64:
		return n, nil
	case uint64:
		if n <= math.MaxInt64 {
			return int64(n), nil
		}
	}
	return 0, errors.New("expected integer")
}

func binaryValue(v any) ([]byte, error) {
	switch b := v.(type) {
	case string:
		return []byte(b), nil
	case []byte:
		return b, nil
	case nil:
		return nil, nil
	default:
		return nil, errors.New("expected binary value")
	}
}

func EmptyTable(value any) bool {
	switch table := value.(type) {
	case []any:
		return len(table) == 0
	case map[any]any:
		return len(table) == 0 || len(table) == 1 && table["_n"] == int64(0)
	}
	return false
}

func (c Call) Text(index, limit int) (string, error) {
	if index >= len(c.Args) {
		return "", fmt.Errorf("%s missing argument %d", c.Name, index)
	}
	s, ok := c.Args[index].(string)
	if !ok || len(s) > limit {
		return "", fmt.Errorf("%s argument %d is not a valid string", c.Name, index)
	}
	return s, nil
}

func (c Call) BattleID() (model.BattleID, error) {
	if len(c.Args) == 0 {
		return 0, fmt.Errorf("%s missing battle ID", c.Name)
	}
	id, err := Integer(c.Args[0])
	if err != nil || id <= 0 || id > math.MaxInt32 {
		return 0, fmt.Errorf("%s invalid battle ID", c.Name)
	}
	return model.BattleID(id), nil
}

func normalize(value any) (any, error) {
	switch v := value.(type) {
	case map[any]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			var name string
			switch k := key.(type) {
			case string:
				name = k
			case int64:
				name = strconv.FormatInt(k, 10)
			case uint64:
				name = strconv.FormatUint(k, 10)
			default:
				name = fmt.Sprint(key)
			}
			if _, exists := out[name]; exists {
				return nil, errors.New("ambiguous dictionary key")
			}
			next, err := normalize(item)
			if err != nil {
				return nil, err
			}
			out[name] = next
		}
		return out, nil
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			next, err := normalize(item)
			if err != nil {
				return nil, err
			}
			out[i] = next
		}
		return out, nil
	default:
		return value, nil
	}
}

func (c Call) Record(index int, target any) error {
	if index >= len(c.Args) {
		return fmt.Errorf("%s missing dictionary %d", c.Name, index)
	}
	argument := c.Args[index]
	if EmptyTable(argument) {
		argument = map[any]any{}
	}
	if _, ok := argument.(map[any]any); !ok {
		return fmt.Errorf("%s argument %d must be a dictionary", c.Name, index)
	}
	value, err := normalize(argument)
	if err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func (c Call) Number(index int, minimum, maximum int64) (int64, error) {
	if index >= len(c.Args) {
		return 0, fmt.Errorf("%s missing argument %d", c.Name, index)
	}
	n, err := Integer(c.Args[index])
	if err != nil || n < minimum || n > maximum {
		return 0, fmt.Errorf("%s invalid integer argument %d", c.Name, index)
	}
	return n, nil
}

func (c Call) Integers(index, limit int, minimum, maximum int64) ([]int, error) {
	if index >= len(c.Args) {
		return nil, errors.New("missing list")
	}
	list, ok := c.Args[index].([]any)
	if !ok || len(list) == 0 || len(list) > limit {
		return nil, errors.New("invalid integer list")
	}
	result := make([]int, 0, len(list))
	seen := make(map[int64]bool, len(list))
	for _, value := range list {
		n, err := Integer(value)
		if err != nil || n < minimum || n > maximum || seen[n] {
			return nil, errors.New("invalid or duplicate integer")
		}
		seen[n] = true
		result = append(result, int(n))
	}
	return result, nil
}

func (c Call) InstanceID(index int) (model.InstanceID, error) {
	if index >= len(c.Args) {
		return "", errors.New("missing instance ID")
	}
	value, ok := c.Args[index].(string)
	if !ok {
		n, err := c.Number(index, 1, math.MaxInt64)
		return model.InstanceID(strconv.FormatInt(n, 10)), err
	}
	if len(value) == 0 || len(value) > 64 {
		return "", errors.New("invalid instance ID")
	}
	for _, char := range value {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '-' || char == '_' {
			continue
		}
		return "", errors.New("invalid instance ID")
	}
	return model.InstanceID(value), nil
}

func (c Call) InstanceIDs(index, limit int, allowEmpty bool) ([]model.InstanceID, error) {
	if index >= len(c.Args) {
		return nil, errors.New("missing equipment selection")
	}
	if allowEmpty && EmptyTable(c.Args[index]) {
		return []model.InstanceID{}, nil
	}
	values, ok := c.Args[index].([]any)
	if !ok || len(values) > limit || !allowEmpty && len(values) == 0 {
		return nil, errors.New("invalid equipment selection")
	}
	list := Call{Name: c.Name, Args: values}
	result := make([]model.InstanceID, 0, len(values))
	seen := map[model.InstanceID]bool{}
	for i := range values {
		id, err := list.InstanceID(i)
		if err != nil || seen[id] {
			return nil, errors.New("invalid or duplicate equipment selection")
		}
		seen[id] = true
		result = append(result, id)
	}
	return result, nil
}

func (c Call) HeroList(index, limit int) ([]model.HeroID, error) {
	if index >= len(c.Args) {
		return nil, errors.New("missing hero list")
	}
	if EmptyTable(c.Args[index]) {
		return []model.HeroID{}, nil
	}
	values, ok := c.Args[index].([]any)
	if !ok || len(values) > limit {
		return nil, errors.New("invalid hero list")
	}
	result := make([]model.HeroID, 0, len(values))
	seen := make(map[model.HeroID]bool, len(values))
	for _, value := range values {
		id, err := Integer(value)
		if err != nil || id < 0 || id > math.MaxInt32 || id != 0 && seen[model.HeroID(id)] {
			return nil, errors.New("invalid or duplicate hero")
		}
		seen[model.HeroID(id)] = true
		result = append(result, model.HeroID(id))
	}
	return result, nil
}

func (c Call) Strings(index, limit, length int) ([]string, error) {
	if index >= len(c.Args) {
		return nil, errors.New("missing string list")
	}
	if EmptyTable(c.Args[index]) {
		return []string{}, nil
	}
	values, ok := c.Args[index].([]any)
	if !ok || len(values) > limit {
		return nil, errors.New("invalid string list")
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		text, ok := value.(string)
		if !ok || len(text) > length || !ValidText(text) {
			return nil, errors.New("invalid string list entry")
		}
		result = append(result, text)
	}
	return result, nil
}

func (c Call) ItemQuantities(index int) (map[model.ItemID]int64, error) {
	if index >= len(c.Args) {
		return nil, errors.New("missing item quantities")
	}
	if EmptyTable(c.Args[index]) {
		return map[model.ItemID]int64{}, nil
	}
	values, ok := c.Args[index].(map[any]any)
	if !ok || len(values) > 100 {
		return nil, errors.New("invalid item quantities")
	}
	result := make(map[model.ItemID]int64, len(values))
	for key, value := range values {
		quantity, err := Integer(value)
		if (key == "__placeholder" || key == "_n") && quantity == 0 && err == nil {
			continue
		}
		if err != nil {
			return nil, errors.New("invalid item quantity")
		}
		var id int64
		if name, text := key.(string); text {
			id, err = strconv.ParseInt(name, 10, 32)
		} else {
			id, err = Integer(key)
		}
		if err != nil || id <= 0 || id > math.MaxInt32 || quantity <= 0 || quantity > 1000000 || result[model.ItemID(id)] != 0 {
			return nil, errors.New("invalid item quantity")
		}
		result[model.ItemID(id)] = quantity
	}
	return result, nil
}

func (c Call) Arguments(count int) error {
	if len(c.Args) != count {
		return fmt.Errorf("invalid %s argument count", c.Name)
	}
	return nil
}

func ValidText(text string) bool { return utf8.ValidString(text) && !strings.ContainsRune(text, 0) }

func (c Call) Player(minimum int64) (model.UserID, error) {
	id, err := c.Number(0, minimum, math.MaxInt64)
	return model.UserID(id), err
}
