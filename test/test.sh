#!/bin/bash

set -e -o pipefail

#	1k.tsv
#	ID	Date of Birth	First Name	Surname	Country

function assert_equal() {
	result=$(diff --brief $1 $2)
	if [ -n "$result" ]; then
		echo "failed: $1 != $2"
	fi
}

# Basic keys
./keycut -k "First Name,Country" 1k.tsv > t1
cut -f 3,5 1k.tsv > t2
assert_equal t1 t2

# Regex
./keycut -e '.*Name
.*ountry' 1k.tsv > t3
assert_equal t3 t1

# Test output delimiters
./keycut -k "First Name,Country" --output-delimiter "|" 1k.tsv > t4
cut -f 3,5 --output-delimiter "|" 1k.tsv > t5
assert_equal t4 t5

# Translate back to original
./keycut -k "First Name,Country" -d "|" --output-delimiter="\t" t4 > t6
assert_equal t6 t1

rm t{1..6}
echo "Tests passed"
