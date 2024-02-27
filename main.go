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
	var flag_complement bool
	var flag_d_input_delim string
	var flag_output_delim string
	var flag_k_key_list string
	var flag_e_regex_list string
	var flag_z_delim bool
	var flag_h_help bool
	var flag_psrv_order bool
	var flag_s_only_delim bool

	// Args common to cut
	flag.StringVar(&flag_d_input_delim, "d", "\t", "Field delimiter")
	flag.BoolVar(&flag_complement, "complement", false, "Print only fields which were not selected")
	flag.StringVar(&flag_output_delim, "output-delimiter", "", "Output field separator (default to input delimiter)")
	flag.BoolVar(&flag_z_delim, "z", false, "Use the null byte as the output line delimiter")
	flag.BoolVar(&flag_s_only_delim, "s", false, "Do not print lines that do not contain the field delimiter")

	// New args
	flag.StringVar(&flag_e_regex_list, "e", "", "Regular expression(s) to select column names. Separate with \\n")
	flag.StringVar(&flag_k_key_list, "k", "", "Key names to select in order of desired output. Separate with ,")
	flag.BoolVar(&flag_h_help, "help", false, "Display help information")
	flag.BoolVar(&flag_psrv_order, "preserve-order", false, "Do not reorder or duplicate columns as listed with -k")

	// Parse args
	flag.Parse()

	if flag_h_help {
		fmt.Fprintf(os.Stderr, "keycut - key based selection of file columns\n\n")
		fmt.Fprintf(os.Stderr, "keycut OPTION... [FILE]...\n\n")
		fmt.Fprintf(os.Stderr, "Print sections from each line of each FILE to standard output.\n")
		fmt.Fprintf(os.Stderr, "If FILE is empty or -, reads from standard input.\n")
		fmt.Fprintf(os.Stderr, "Select using key name with -k or using regular expressions with -e.\n\n")
		flag.Usage()
	}

	if flag_k_key_list != "" && flag_e_regex_list != "" {
		abort(errors.New("may use only one of -k or -e\n try 'keycut --help' for more information"))
	} else if flag_k_key_list == "" && flag_e_regex_list == "" {
		abort(errors.New("must select fields with -k or -e\n try 'keycut --help' for more information"))
	}
	if flag_z_delim {
		line_delim = byte(0)
	}

	// Unescape separator characters
	ifs := unescapeString(flag_d_input_delim)
	var ofs string
	if flag_output_delim == "" {
		ofs = ifs
	} else {
		ofs = unescapeString(flag_output_delim)
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
	case flag_e_regex_list != "":
		checker = MakeRegexChecker(flag_e_regex_list)
		keyIndices = findKeysWithChecker(ifs, ofs, scanner, writer, checker)
	case flag_k_key_list != "":
		// checker = MakeMembershipChecker(fieldBlurb)
		fields := strings.Split(flag_k_key_list, ",")
		keyIndices = findKeysSimple(fields, ifs, ofs, flag_complement, flag_psrv_order, scanner, writer)
	}

	// Process all remaining lines
	processLines(
		keyIndices, 
		[]byte(ifs), 
		[]byte(ofs), 
		flag_s_only_delim,
		scanner, 
		writer)
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
	save_order bool,
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
	} else if save_order {
		for i, k := range keyNames {
			positions[k] = i
		}
		for i, f := range fields {
			_, ok := positions[f]
			if ok {
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
	only_delimited bool,
	scanner *bufio.Scanner,
	writer *bufio.Writer,
) {
	var buffer bytes.Buffer
	for scanner.Scan() {
		fields := bytes.Split(scanner.Bytes(), ifs)
		if len(fields) == 1 && only_delimited {
			continue
		}
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
