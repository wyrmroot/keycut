# keycut
Columnar slicing of large files using pattern matching

# About
`keycut` is inspired by `cut` but focuses on working with tabular files by header names (keys).
It is intended to reduce calls to heavier scripting languages for the sole purpose of file splitting and fit into the coreutils ecosystem of bash pipeline functions.

Like `cut`, it prints columns according to the fields selected and aims to be performant including working with larger-than-memory files.

`keycut` introduces two new ways of selecting columns:
- By key name using `-k`
- By regular expressions using `-e`

All other options from `cut` are reproduced by `keycut`.
The implementation of `-e`should be familiar to users of `grep`.

Like `cut`, slicing on delimiters is done naively (and quickly) without regard for the context of potentially escaped delimiters (such as quoted commas in a CSV). For those cases, we recommend using a proper CSV parser.

# License
`keycut` is distributed under the MIT license and depends only the standard Go library.
