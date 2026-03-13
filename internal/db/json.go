package db

import (
	"encoding/json"

	"codex-mem/internal/domain/common"
)

func marshalStringSlice(values []string) (string, error) {
	if len(values) == 0 {
		return "[]", nil
	}
	body, err := json.Marshal(values)
	if err != nil {
		return "", common.WrapError(common.ErrWriteFailed, "marshal string slice", err)
	}
	return string(body), nil
}

func unmarshalStringSlice(value string) ([]string, error) {
	if value == "" {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal([]byte(value), &values); err != nil {
		return nil, common.WrapError(common.ErrReadFailed, "unmarshal string slice", err)
	}
	return values, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func intToBool(value int) bool {
	return value != 0
}
