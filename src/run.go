package keycut

import (
	"bufio"
	"bytes"
	"flag"
	"os"
)

func Run(file string, args *Options) {
	// Locate stream source
	var scanner *bufio.Scanner
	if file == "-" {
		scanner = bufio.NewScanner(os.Stdin)
	} else {
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

	if args.SelectKeys != nil && !(args.Complement || args.PreserveOrder) {
		// We are reordering values
		keyIndices = SelectOrderedKeyNames(
			args.SelectKeys,
			args.InputDelimiter,
			args.OutputDelimiter,
			args.NewLineDelimiter,
			scanner,
			writer,
		)
	} else if args.SelectExpressions != nil {
		checker = makeRegexChecker(args.SelectExpressions)
		keyIndices = SelectWithChecker(
			args.InputDelimiter,
			args.OutputDelimiter,
			args.NewLineDelimiter,
			args.Complement,
			scanner,
			writer,
			checker,
		)
	} else {
		checker = makeMembershipChecker(args.SelectKeys)
		keyIndices = SelectWithChecker(
			args.InputDelimiter,
			args.OutputDelimiter,
			args.NewLineDelimiter,
			args.Complement,
			scanner,
			writer,
			checker,
		)
	}

	// Process all remaining lines
	processLines(
		keyIndices,
		[]byte(args.InputDelimiter),
		[]byte(args.OutputDelimiter),
		args.OnlyDelimitedLines,
		args.NewLineDelimiter,
		scanner,
		writer,
	)
	writer.Flush()
}

// Prints all lines of the file after the first
func processLines(
	cols []int,
	ifs, ofs []byte,
	only_delimited bool,
	line_delim byte,
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
