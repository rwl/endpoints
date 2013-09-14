package endpoint

import (
	"github.com/rwl/go-endpoints/endpoints"
	"sort"
	"strings"
)

// Get a copy of "methods" sorted the way they would be on the live server.
//
// Args:
//   methods: JSON configuration of an API"s methods.
//
// Returns:
//   The same configuration with the methods sorted based on what order
//   they'll be checked by the server.
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
	sort.Sort(ByPath(sortedMethods))
	return sortedMethods
}

type ByPath []*methodInfo

func (by ByPath) Len() int {
	return len(by)
}

// Less returns whether the element with index i should sort
// before the element with index j.
func (by ByPath) Less(i, j int) bool {
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

func (by ByPath) Swap(i, j int) {
	by[i], by[j] = by[j], by[i]
}

// Calculate the score for this path, used for comparisons.
//
// Higher scores have priority, and if scores are equal, the path text
// is sorted alphabetically.  Scores are based on the number and location
// of the constant parts of the path.  The server has some special handling
// for variables with regexes, which we don't handle here.
//
// Args:
//   path: The request path that we"re calculating a score for.
//
// Returns:
//   The score for the given path.
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
	// Shift by 31 instead of 32 because some (!) versions of Python like
	// to convert the int to a long if we shift by 32, and the sorted()
	// function that uses this blows up if it receives anything but an int.
	score <<= uint(31 - len(parts))
	return score
}
