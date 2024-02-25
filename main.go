package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	re "regexp"
	"strings"
)

var line_delim = byte('\n')

func main() {
	var doComplement bool
	var ifs_raw string
	var ofs_raw string
	var fieldBlurb string
	var regexBlurb string
	var z_delim bool
	var s_only_delim bool

	// Args common to cut
	flag.StringVar(&ifs_raw, "d", "\t", "Field delimiter (default TAB)")
	flag.BoolVar(&doComplement, "complement", false, "Print only fields which were not selected")
	flag.StringVar(&ofs_raw, "output-delimiter", "", "Output field separator (default to input delimiter)")
	flag.BoolVar(&z_delim, "z", false, "Use the null byte as the line delimiter")
	flag.BoolVar(&s_only_delim, "s", false, "Do not print lines that do not contain the field delimiter")

	// New args
	flag.StringVar(&regexBlurb, "e", "", "Regular expression(s) to select column names. Separate with newline.")
	flag.StringVar(&fieldBlurb, "k", "", "Key names to select, in order of desired output. Separate with ','")

	// Parse args
	flag.Parse()
	if fieldBlurb != "" && regexBlurb != "" {
		abort(errors.New("may use only one of -f or -e"))
	} else if fieldBlurb == "" && regexBlurb == "" {
		flag.PrintDefaults()
		abort(errors.New("must specify -f or -e"))
	}
	if z_delim {
		line_delim = byte(0)
	}
	// Unescape separator characters
	ifs := unescapeString(ifs_raw)
	var ofs string
	if ofs_raw == "" {
		ofs = ifs
	} else {
		ofs = unescapeString(ofs_raw)
	}

	// Locate stream source
	var scanner *bufio.Scanner
	switch {
	case flag.NArg() == 0:
		// No file means read STDIN
		fallthrough
	case flag.Arg(0) == "-":
		// Filename - means read STDIN
		scanner = bufio.NewScanner(os.Stdin)
	default:
		// Else, open file
		openFile, err := os.Open(flag.Arg(0))
		if err != nil {
			abort(err)
		}
		defer openFile.Close()
		scanner = bufio.NewScanner(openFile)
	}
	scanner.Split(bufio.ScanLines)
	writer := bufio.NewWriter(os.Stdout)

	// Consume header and locate column positions of interest
	var checker Checker
	var keyIndices []int
	switch {
	case regexBlurb != "":
		checker = MakeRegexChecker(regexBlurb)
		keyIndices = findKeysWithChecker(ifs, ofs, scanner, writer, checker)
	case fieldBlurb != "":
		// checker = MakeMembershipChecker(fieldBlurb)
		fields := strings.Split(fieldBlurb, ",")
		keyIndices = findKeysSimple(fields, ifs, ofs, doComplement, scanner, writer)
	}
	fmt.Printf("got keys %v\n", keyIndices)

	// Process all remaining lines
	processLines(keyIndices, []byte(ifs), []byte(ofs), scanner, writer)
	writer.Flush()
}

/*
A polite exit without a panic traceback
*/
func abort(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func unescapeString(s string) string {
	r := strings.NewReplacer(
		"\\t", "\t",
		"\\n", "\n",
		"\\r", "\r",
		"\\b", "\b",
		"\\a", "\a",
		"\\f", "\f",
		"\\v", "\v",
		"\\\\", "\\",
	)
	return r.Replace(s)
}

/*
Signature of a function which validates whether a value is a member of a set, and if so what
index position it should occupy.

If the checker has no opinion about the position of the value, it should return (-1, true).
*/
type Checker func(value string) (position int, found bool)

/*
A closure which remembers a set of comma-separated values in blurb and provides a Checker
for if an item is a member of that set.
*/
func MakeMembershipChecker(blurb string) Checker {
	fields := strings.Split(blurb, ",")
	positions := map[string]int{}
	for i, v := range fields {
		positions[v] = i
	}
	return func(value string) (int, bool) {
		pos, ok := positions[value]
		return pos, ok
	}
}

/*
A closure which remembers a set of newline-separated regex patterns and provides a Checker
for if a future item is matched by any pattern. Does not consider positionality of results.
*/
func MakeRegexChecker(blurb string) Checker {
	raw_patterns := strings.Split(blurb, "\n")
	expressions := make([]*re.Regexp, len(raw_patterns))
	for i, raw_pat := range raw_patterns {
		var err error
		expressions[i], err = re.Compile(raw_pat)
		if err != nil {
			abort(err)
		}
	}

	return func(value string) (int, bool) {
		for _, exp := range expressions {
			if exp.Match([]byte(value)) {
				return -1, true
			}
		}
		return -1, false
	}
}

/*
Searches for the index location of columns whose key matches the rules contained by
the checker. Returns a slice of those locations.
*/
func findKeysWithChecker(
	ifs, ofs string,
	scanner *bufio.Scanner,
	writer *bufio.Writer,
	checker Checker,
) []int {
	// Scan just one line
	scanner.Scan()
	fields := strings.Split(scanner.Text(), ifs)
	results := make([]int, len(fields))
	for i := range results {
		results[i] = -1
	}
	for i, f := range fields {
		pos, ok := checker(f)
		if ok {
			if pos > -1 {
				results[pos] = i
			} else {
				results[i] = i
			}
		}
	}

	// Remove missing items from results,
	// and print the final header line
	var buffer bytes.Buffer
	clean_results := make([]int, 0, len(results))
	for i, r := range results {
		if r > -1 {
			clean_results = append(clean_results, r)
			buffer.WriteString(fields[r])
			if i < len(results)-1 {
				buffer.WriteString(ofs)
			}
		}
	}
	buffer.WriteByte(line_delim)
	buffer.WriteTo(writer)
	return clean_results
}

/*
Searches for the index location of columns whose key matches the name of an item in
keyNames. Returns a slice of those locations.
*/
func findKeysSimple(
	keyNames []string,
	ifs, ofs string,
	invert bool,
	scanner *bufio.Scanner,
	writer *bufio.Writer,
) []int {
	// Create as many results as requested column names
	// (may exceed table width, if repeats are desired)
	results := make([]int, 0, len(keyNames))
	var buffer bytes.Buffer

	// Scan one line and remember the position of the keys
	scanner.Scan()
	fields := strings.Split(scanner.Text(), ifs)
	positions := map[string]int{}

	// If taking complement, iterate over the original headers and confirm
	// lack of membership in keyNames
	if invert {
		for i, k := range keyNames {
			positions[k] = i
		}
		for i, f := range fields {
			_, ok := positions[f]
			if !ok {
				results = append(results, i)
				buffer.WriteString(f)
				if i < len(fields)-1 {
					buffer.WriteString(ofs)
				}
			}
		}
	} else {
		// Else, iterate over keyNames and find their positions in headers
		for i, f := range fields {
			positions[f] = i
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
	}
	// Write the header line
	buffer.WriteByte(line_delim)
	buffer.WriteTo(writer)
	return results
}

// Prints all lines of the file after the first
func processLines(
	cols []int,
	ifs, ofs []byte,
	scanner *bufio.Scanner,
	writer *bufio.Writer,
) {
	var buffer bytes.Buffer
	for scanner.Scan() {
		fields := bytes.Split(scanner.Bytes(), ifs)
		for i, pos := range cols {
			if pos < len(fields) {
				buffer.Write(fields[pos])
			}
			if i < len(cols)-1 {
				buffer.Write(ofs)
			}
		}
		buffer.WriteByte(line_delim)
		buffer.WriteTo(writer)
	}
}
