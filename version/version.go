/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package version

import (
	"fmt"
)

// ApplicationVersion represents application version.
var ApplicationVersion = NewVersion(0, 4, 0)

// SemanticVersion represents version information with Semantic Versioning specifications.
type SemanticVersion struct {
	major uint
	minor uint
	patch uint
}

// NewVersion initializes a new instance of SemanticVersion.
func NewVersion(major, minor, patch uint) *SemanticVersion {
	return &SemanticVersion{
		major: major,
		minor: minor,
		patch: patch,
	}
}

// String returns a string that represents this instance.
func (v *SemanticVersion) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.major, v.minor, v.patch)
}

// IsEqual returns true if version instance is equal to the second.
func (v *SemanticVersion) IsEqual(v2 *SemanticVersion) bool {
	if v == v2 {
		return true
	}
	return v.major == v2.major && v.minor == v2.minor && v.patch == v2.patch
}

// IsLess returns true if version instance is less than the second.
func (v *SemanticVersion) IsLess(v2 *SemanticVersion) bool {
	if v == v2 {
		return false
	}
	if v.major == v2.major {
		if v.minor == v2.minor {
			if v.patch == v2.patch {
				return false
			}
			return v.patch < v2.patch
		}
		return v.minor < v2.minor
	}
	return v.major < v2.major
}

// IsLessOrEqual returns true if version instance is less than or equal to the second.
func (v *SemanticVersion) IsLessOrEqual(v2 *SemanticVersion) bool {
	return v.IsLess(v2) || v.IsEqual(v2)
}

// IsGreater returns true if version instance is greater than the second.
func (v *SemanticVersion) IsGreater(v2 *SemanticVersion) bool {
	if v == v2 {
		return false
	}
	if v.major == v2.major {
		if v.minor == v2.minor {
			if v.patch == v2.patch {
				return false
			}
			return v.patch > v2.patch
		}
		return v.minor > v2.minor
	}
	return v.major > v2.major
}

// IsGreaterOrEqual returns true if version instance is greater than or equal to the second.
func (v *SemanticVersion) IsGreaterOrEqual(v2 *SemanticVersion) bool {
	return v.IsGreater(v2) || v.IsEqual(v2)
}
