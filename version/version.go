/*
 * Copyright (c) 2017-2018 Miguel Ángel Ortuño.
 * See the COPYING file for more information.
 */

package version

import (
	"fmt"
)

var Version *SemanticVersion = NewVersion(0, 5, 0)

type SemanticVersion struct {
	major uint
	minor uint
	patch uint
}

func NewVersion(major, minor, patch uint) *SemanticVersion {
	return &SemanticVersion{
		major: major,
		minor: minor,
		patch: patch,
	}
}

func (v *SemanticVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
}

func (v *SemanticVersion) IsEqual(v2 *SemanticVersion) bool {
	if v == v2 {
		return true
	}
	return v.major == v2.major && v.minor == v2.minor && v.patch == v2.patch
}

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

func (v *SemanticVersion) IsLessOrEqual(v2 *SemanticVersion) bool {
	return v.IsLess(v2) || v.IsEqual(v2)
}

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

func (v *SemanticVersion) IsGreaterOrEqual(v2 *SemanticVersion) bool {
	return v.IsGreater(v2) || v.IsEqual(v2)
}
