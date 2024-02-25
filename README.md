# keycut
Columnar file slicing using pattern matching

`keycut` is inspired by `cut` but with a focus on working with tabular files by header names (keys).
Like `cut`, it prints columns according to the fields selected and aims to be performant including working with larger-than-memory files.

`keycut` selects fields in two ways:
- By key name using `-k`
- By regular expressions using `-e`

Most options from `cut` are available, including delimiter specification and complement printing.

# TODO
- [ ] field selection by key
- [ ] field selection by regex
- [ ] ranges
- [ ] case insensitivity
- [x] complement
- [ ] Test with "bad" files (mismatched col numbers / missing delimiters)
- [ ] Test with larger-than-memory files
- [x] Wrap mustcompile w/ stderr