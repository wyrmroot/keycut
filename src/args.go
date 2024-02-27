package keycut

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

type Options struct {
	InputDelimiter     string
	OutputDelimiter    string
	NewLineDelimiter   byte
	Complement         bool
	PreserveOrder      bool
	Help               bool
	OnlyDelimitedLines bool
	SelectKeys         []string
	SelectExpressions  []string
	Filepaths          []string
}

func ParseOptions() *Options {
	// Prepare receivers for args
	ArgOptions := Options{
		NewLineDelimiter: byte('\n'),
	}
	var flag_z bool
	var flag_d string
	var flag_k string
	var flag_e string
	var flag_output_delim string

	// Args common to cut
	flag.StringVar(&flag_d, "d", "\t", "Field delimiter")
	flag.BoolVar(&ArgOptions.Complement, "complement", false, "Print only fields which were not selected")
	flag.StringVar(&flag_output_delim, "output-delimiter", "", "Output field separator (default to input delimiter)")
	flag.BoolVar(&flag_z, "z", false, "Use the null byte as the output line delimiter")
	flag.BoolVar(&ArgOptions.OnlyDelimitedLines, "s", false, "Do not print lines that do not contain the field delimiter")

	// New args
	flag.StringVar(&flag_e, "e", "", "Regular expression(s) to select column names. Separate with \\n")
	flag.StringVar(&flag_k, "k", "", "Key names to select in order of desired output. Separate with ,")
	flag.BoolVar(&ArgOptions.Help, "help", false, "Display help information")
	flag.BoolVar(&ArgOptions.PreserveOrder, "preserve-order", false, "Do not reorder or duplicate columns as listed with -k")

	// Parse args
	flag.Parse()

	// Early exit for help
	if ArgOptions.Help {
		return &ArgOptions
	}
	if ArgOptions.Help {
		fmt.Fprintf(os.Stderr, "keycut - key based selection of file columns\n\n")
		fmt.Fprintf(os.Stderr, "keycut OPTION... [FILE]...\n\n")
		fmt.Fprintf(os.Stderr, "Print sections from each line of each FILE to standard output.\n")
		fmt.Fprintf(os.Stderr, "If FILE is empty or -, reads from standard input.\n")
		fmt.Fprintf(os.Stderr, "Select using key name with -k or using regular expressions with -e.\n\n")
		flag.Usage()
		os.Exit(0)
	}

	// Arg validation
	if flag_k != "" {
		if flag_e != "" {
			abort(errors.New("may use only one of -k or -e\ntry 'keycut --help' for more information"))
		}
		ArgOptions.SelectKeys = strings.Split(flag_k, ",")
	} else if flag_e != "" {
		ArgOptions.SelectExpressions = strings.Split(flag_e, "\n")
	} else {
		abort(errors.New("must select fields with -k or -e\ntry 'keycut --help' for more information"))
	}

	if flag_z {
		ArgOptions.NewLineDelimiter = byte(0)
	}

	ArgOptions.InputDelimiter = unescapeString(flag_d)
	if flag_output_delim == "" {
		ArgOptions.OutputDelimiter = ArgOptions.InputDelimiter
	} else {
		ArgOptions.OutputDelimiter = unescapeString(flag_output_delim)
	}
	if flag.NArg() > 0 {
		ArgOptions.Filepaths = flag.Args()
	} else {
		// Indicate to use Stdin
		ArgOptions.Filepaths = append(ArgOptions.Filepaths, "-")
	}

	return &ArgOptions
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
