// Copyright 2022 The jackal Authors
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

package stringmatcher

import (
	"regexp"
)

// RegexMatcher implements regular expression string matcher.
type RegexMatcher struct {
	regex *regexp.Regexp
}

// NewRegExMatcher returns a new initialized RegexMatcher.
func NewRegExMatcher(expr string) (*RegexMatcher, error) {
	regex, err := regexp.Compile(expr)
	if err != nil {
		return nil, err
	}
	return &RegexMatcher{regex: regex}, nil
}

// Matches returns true if str matches em regular expression.
func (em *RegexMatcher) Matches(str string) bool {
	return em.regex.MatchString(str)
}
