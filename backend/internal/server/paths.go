package server

import (
	"errors"
	"path/filepath"
	"strings"
)

// JoinUnder returns filepath.Join(base, rel) only if the result stays under base.
func JoinUnder(base string, rel string) (string, error) {
	base = filepath.Clean(base)
	full := filepath.Join(base, rel)
	full = filepath.Clean(full)
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	fullAbs, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	sep := string(filepath.Separator)
	if fullAbs != baseAbs && !strings.HasPrefix(fullAbs, baseAbs+sep) {
		return "", errors.New("path escapes templates root")
	}
	return fullAbs, nil
}
