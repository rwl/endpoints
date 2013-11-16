// Copyright 2013 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package endpoint

import "fmt"

// braceIndices returns the first level curly brace indices from a string.
// It returns an error in case of unbalanced braces.
//
// Copied from github.com/gorilla/mux/regexp.go
func braceIndices(s string) ([]int, error) {
	var level, idx int
	idxs := make([]int, 0)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{':
			if level++; level == 1 {
				idx = i
			}
		case '}':
			if level--; level == 0 {
				idxs = append(idxs, idx, i+1)
			} else if level < 0 {
				return nil, fmt.Errorf("unbalanced braces in %q", s)
			}
		}
	}
	if level != 0 {
		return nil, fmt.Errorf("unbalanced braces in %q", s)
	}
	return idxs, nil
}
