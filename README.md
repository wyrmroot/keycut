# keycut
Columnar slicing of large files using pattern matching

## About
`keycut` is inspired by `cut` but focuses on working with tabular files by header names (keys).
It is intended to fit into the coreutils ecosystem of bash pipeline functions and reduce calls to heavier scripting languages for file splitting.

Like `cut`, it prints columns according to the fields selected and aims to be performant including working with larger-than-memory files.

`keycut` introduces two new ways of selecting columns:
- By key name using `-k`
- By regular expressions using `-e`

All other options from `cut` are reproduced by `keycut`.
The implementation of `-e`should be familiar to users of `grep`.

Like `cut`, slicing on delimiters is done naively (and quickly) without regard for the context of potentially escaped delimiters (such as quoted commas in a CSV). For those cases, we recommend using a more comprehensive CSV parser.

## Usage
`-k <key-names>`

Selects columns by exact key, where key-names is a comma separated list of strings.
May not be used with another selector.
Note that columns are printed in the order they are listed.

`-e <expressions>`

Selects columns by regular expression where multiple expressions are separated by a newline character `\n`.
Columns with a header satisfying any of the patterns will be printed.
May not be used with another selector.

`--complement` 

After fields are selected, this setting instructs the program to print all columns which did *not* match the selection.

`-d <delimiter>` 

Input field delimiter, the character on which to split columns (default `\t`).

`--help`

Displays help information.

`-output-delimiter` 

Output field delimiter, the character or string which will be used to separate columns as they are printed. Defaults to match the input delimiter.

`--preserve-order`

Ensures that columns are printed in the order they appeared in the original file. Otherwise, they are printed according to the order they were requested with `-k` (including duplication). Has no effect on unordered selection methods such as `-e`.

`-s`

Do not print lines which do not contain the field delimiter.
The default behavior (preserved from `cut`) is to print the entirety of lines which do not contain the delimiter.

`-z`

Use a zero byte as the output line delimiter (instead of `\n`).
Used mainly alongside other commands which may do the same to enforce filename compatibility.

## License
`keycut` is distributed under the MIT license and depends only on the standard Go library.

## Performance ideas
- producer/consumer goroutines for read/write
- Manual GC