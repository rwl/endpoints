// Copyright 2013 Google Inc. All Rights Reserved.
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

package endpoints_server


import (
	"github.com/rwl/go-endpoints/endpoints"
	"sort"
	"strings"
)

// Returns the same method configurations sorted based on the order
// they'll be checked by the server.
func sortMethods(methods map[string]*endpoints.ApiMethod) []*methodInfo {
	if methods == nil {
		return nil
	}
	sortedMethods := make([]*methodInfo, len(methods))
	i := 0
	for name, m := range methods {
		sortedMethods[i] = &methodInfo{name, m}
		i++
	}
	sort.Sort(byPath(sortedMethods))
	return sortedMethods
}

type byPath []*methodInfo

func (by byPath) Len() int {
	return len(by)
}

// Returns whether the element with index i should sort
// before the element with index j.
func (by byPath) Less(i, j int) bool {
	methodInfo1 := by[i].apiMethod
	methodInfo2 := by[j].apiMethod

	path1 := methodInfo1.Path
	path2 := methodInfo2.Path

	pathScore1 := scorePath(path1)
	pathScore2 := scorePath(path2)
	if pathScore1 != pathScore2 {
		// Higher path scores come first.
		return pathScore1 > pathScore2
	}

	// Compare by path text next, sorted alphabetically.
	if path1 != path2 {
		return path1 < path2
	}

	// All else being equal, sort by HTTP method.
	httpMethod1 := methodInfo1.HttpMethod
	httpMethod2 := methodInfo2.HttpMethod
	return httpMethod1 < httpMethod2
}

func (by byPath) Swap(i, j int) {
	by[i], by[j] = by[j], by[i]
}

// Calculate the score for this path, used for comparisons.
//
// Higher scores have priority, and if scores are equal, the path text
// is sorted alphabetically.  Scores are based on the number and location
// of the constant parts of the path.  The server has some special handling
// for variables with regexes, which we don't handle here.
func scorePath(path string) int {
	score := 0
	parts := strings.Split(path, "/")
	for _, part := range parts {
		score <<= 1
		if part == "" || !strings.HasPrefix(part, "{") {
			// Found a constant.
			score += 1
		}
	}
	score <<= uint(31 - len(parts)) // todo: shift by 32 instead of 31 (?)
	return score
}
