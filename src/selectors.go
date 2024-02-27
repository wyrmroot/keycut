package keycut

import (
	"bufio"
	"bytes"
	re "regexp"
	"strings"
)

/*
Searches for the index location of columns whose key matches the rules contained by
the checker. Returns a slice of those locations.
*/
func SelectWithChecker(
	ifs, ofs string,
	line_delim byte,
	invert bool,
	scanner *bufio.Scanner,
	writer *bufio.Writer,
	checker Checker,
) []int {
	scanner.Scan()
	headers := strings.Split(scanner.Text(), ifs)
	results := make([]int, 0, len(headers))

	for i, h := range headers {
		ok := checker(h)
		if (ok && !invert) || (!ok && invert) {
			results = append(results, i)
		}
	}

	// Print the final header line
	var buffer bytes.Buffer
	for i, r := range results {
		buffer.WriteString(headers[r])
		if i < len(results)-1 {
			buffer.WriteString(ofs)
		}
	}
	buffer.WriteByte(line_delim)
	buffer.WriteTo(writer)
	return results
}

/*
Searches for the index location of columns whose key matches the name of an item in
keyNames. Returns a slice of those locations.
*/
func SelectOrderedKeyNames(
	keyNames []string,
	ifs, ofs string,
	line_delim byte,
	scanner *bufio.Scanner,
	writer *bufio.Writer,
) []int {
	// Create as many results as requested column names
	// (may exceed table width, if repeats are desired)
	results := make([]int, 0, len(keyNames))
	var buffer bytes.Buffer

	// Scan one line and remember the position of the keys
	scanner.Scan()
	headers := strings.Split(scanner.Text(), ifs)
	positions := map[string]int{}

	// Iterate over keyNames and find their positions in headers
	for i, h := range headers {
		positions[h] = i
	}
	for i, k := range keyNames {
		pos, ok := positions[k]
		if ok {
			results = append(results, pos)
			buffer.WriteString(k)
			if i < len(keyNames)-1 {
				buffer.WriteString(ofs)
			}
		}
	}

	// Write the header line
	buffer.WriteByte(line_delim)
	buffer.WriteTo(writer)
	return results
}

/*
Signature of a function which validates whether a value is a member of a set.
*/
type Checker func(value string) (found bool)

/*
A closure which remembers a set of comma-separated values in blurb and provides a Checker
for if an item is a member of that set.
*/
func makeMembershipChecker(fields []string) Checker {
	set := map[string]int{}
	for i, v := range fields {
		set[v] = i
	}
	return func(value string) bool {
		_, ok := set[value]
		return ok
	}
}

/*
A closure which remembers a set of newline-separated regex patterns and provides a Checker
for if a future item is matched by any pattern. Does not consider positionality of results.
*/
func makeRegexChecker(raw_patterns []string) Checker {
	expressions := make([]*re.Regexp, len(raw_patterns))
	for i, raw_pat := range raw_patterns {
		var err error
		expressions[i], err = re.Compile(raw_pat)
		if err != nil {
			abort(err)
		}
	}

	return func(value string) bool {
		for _, exp := range expressions {
			if exp.Match([]byte(value)) {
				return true
			}
		}
		return false
	}
}
