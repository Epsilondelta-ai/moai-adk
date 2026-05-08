package bridge

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

func sessionStartedAtFromSessionFile(path string) int64 {
	name := filepath.Base(path)
	if idx := strings.Index(name, "_"); idx > 0 {
		name = name[:idx]
	}
	if len(name) < len("2006-01-02T15-04-05-000Z") {
		return 0
	}
	candidate := name[:len("2006-01-02T15-04-05-000Z")]
	parts := strings.Split(strings.TrimSuffix(candidate, "Z"), "T")
	if len(parts) != 2 {
		return 0
	}
	timeParts := strings.Split(parts[1], "-")
	if len(timeParts) != 4 {
		return 0
	}
	iso := fmt.Sprintf("%sT%s:%s:%s.%sZ", parts[0], timeParts[0], timeParts[1], timeParts[2], timeParts[3])
	if parsed, err := time.Parse(time.RFC3339Nano, iso); err == nil {
		return parsed.UTC().UnixMilli()
	}
	return 0
}

func int64FromAny(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	case jsonNumber:
		parsed, err := typed.Int64()
		if err == nil {
			return parsed
		}
	case string:
		if parsed, err := time.Parse(time.RFC3339, typed); err == nil {
			return parsed.UTC().UnixMilli()
		}
	}
	return 0
}

type jsonNumber interface {
	Int64() (int64, error)
}
