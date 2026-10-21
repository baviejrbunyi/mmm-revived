package patches

import (
	"path"
	"strings"
)

const (
	Executable            = "Marvel MM CN.exe"
	MaxFileSize     int64 = 512 << 20
	MaxManifestSize int64 = 1 << 20
	MaxFiles              = 32
)

const stateDirectory = ".mmm-revived"

func ValidHash(value string) bool {
	return len(value) == 64 && !strings.ContainsFunc(value, func(r rune) bool {
		return (r < '0' || r > '9') && (r < 'a' || r > 'f')
	})
}

func ValidPath(value string) bool {
	if value == "" || value == "." || strings.ContainsAny(value, "\\:<>\"|?*") || path.IsAbs(value) || path.Clean(value) != value || strings.ContainsFunc(value, func(r rune) bool { return r < 32 }) {
		return false
	}
	for part := range strings.SplitSeq(value, "/") {
		if part == ".." || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") || reservedDeviceName(part) {
			return false
		}
	}
	lower := strings.ToLower(value)
	return lower != stateDirectory && !strings.HasPrefix(lower, stateDirectory+"/")
}

func reservedDeviceName(part string) bool {
	base, _, _ := strings.Cut(part, ".")
	base = strings.ToUpper(base)
	switch base {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	return len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '0' && base[3] <= '9'
}
