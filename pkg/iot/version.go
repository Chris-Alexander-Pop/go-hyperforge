package iot

import (
	"strings"

	"golang.org/x/mod/semver"
)

// NormalizeVersion ensures a version string is suitable for golang.org/x/mod/semver.
// Bare versions like "1.2.3" become "v1.2.3". Already-prefixed and empty strings
// are returned as-is (empty remains invalid).
func NormalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return v
	}
	if strings.HasPrefix(v, "v") || strings.HasPrefix(v, "V") {
		return "v" + strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V")
	}
	return "v" + v
}

// IsValidVersion reports whether v is a valid semantic version (with or without "v" prefix).
func IsValidVersion(v string) bool {
	return semver.IsValid(NormalizeVersion(v))
}

// CompareVersions compares two semantic versions.
// Returns -1 if a < b, 0 if a == b, +1 if a > b.
// Both versions may omit the leading "v".
// Returns an error if either version is invalid.
func CompareVersions(a, b string) (int, error) {
	na, nb := NormalizeVersion(a), NormalizeVersion(b)
	if !semver.IsValid(na) {
		return 0, ErrInvalidVersion(a, nil)
	}
	if !semver.IsValid(nb) {
		return 0, ErrInvalidVersion(b, nil)
	}
	return semver.Compare(na, nb), nil
}

// IsNewerVersion reports whether available is a newer semantic version than current.
func IsNewerVersion(available, current string) (bool, error) {
	cmp, err := CompareVersions(available, current)
	if err != nil {
		return false, err
	}
	return cmp > 0, nil
}
