// Copyright 2020 The jackal Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package version

import (
	"fmt"
)

// Version represents application version.
var Version = NewVersion(0, 53, 0)

// APIVersion represents admin API version.
var APIVersion = NewVersion(1, 0, 0)

// ClusterAPIVersion represents cluster API version.
var ClusterAPIVersion = NewVersion(1, 0, 0)

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

// Major returns version major value.
func (v *SemanticVersion) Major() uint { return v.major }

// Minor returns version minor value.
func (v *SemanticVersion) Minor() uint { return v.minor }

// Patch returns version patch value.
func (v *SemanticVersion) Patch() uint { return v.patch }

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
