/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package util

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
