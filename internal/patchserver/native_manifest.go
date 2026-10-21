package patchserver

import (
	"errors"

	"mmm-revived/internal/json"
)

type nativePatchListFields NativePatchList

type nativeManifestFiles map[string]map[string]map[string]json.RawMessage

func (list NativePatchList) MarshalJSON() ([]byte, error) {
	files := make(nativeManifestFiles, len(list.Files))
	for origin, versions := range list.Files {
		files[origin] = make(map[string]map[string]json.RawMessage, len(versions))
		for version, entries := range versions {
			values := make(map[string]json.RawMessage, len(entries)+1)
			for name, info := range entries {
				if name == "restart" {
					return nil, errors.New("restart is reserved for native version metadata")
				}
				value, err := json.Marshal(info)
				if err != nil {
					return nil, err
				}
				values[name] = value
			}
			if list.RestartVersions[version] {
				values["restart"] = json.RawMessage("true")
			}
			files[origin][version] = values
		}
	}
	return json.Marshal(struct {
		nativePatchListFields
		Files nativeManifestFiles `json:"files"`
	}{nativePatchListFields(list), files})
}

func (list *NativePatchList) UnmarshalJSON(data []byte) error {
	var value struct {
		nativePatchListFields
		Files nativeManifestFiles `json:"files"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	result := NativePatchList(value.nativePatchListFields)
	result.Files = make(map[string]map[string]map[string]NativeFileInfo, len(value.Files))
	result.RestartVersions = make(map[string]bool)
	for origin, versions := range value.Files {
		result.Files[origin] = make(map[string]map[string]NativeFileInfo, len(versions))
		for version, entries := range versions {
			files := make(map[string]NativeFileInfo, len(entries))
			for name, raw := range entries {
				if name == "restart" {
					var restart *bool
					if err := json.Unmarshal(raw, &restart); err != nil || restart == nil {
						return errors.New("native restart metadata must be boolean")
					}
					if previous, exists := result.RestartVersions[version]; exists && previous != *restart {
						return errors.New("conflicting native restart metadata")
					}
					result.RestartVersions[version] = *restart
					continue
				}
				var info NativeFileInfo
				if err := json.Unmarshal(raw, &info); err != nil {
					return err
				}
				files[name] = info
			}
			result.Files[origin][version] = files
		}
	}
	*list = result
	return nil
}
