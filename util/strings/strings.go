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

package stringsutil

// SplitKeyAndValue splits a string between 'key' and 'value' sub elements.
func SplitKeyAndValue(str string, sep byte) (key string, value string) {
	j := -1
	for i := 0; i < len(str); i++ {
		if str[i] == sep {
			j = i
			break
		}
	}
	if j == -1 {
		return "", ""
	}
	key = str[0:j]
	value = str[j+1:]
	return
}

// StringSliceContains returns true in case str is contained into slice.
func StringSliceContains(str string, slice []string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
