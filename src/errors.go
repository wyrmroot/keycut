package keycut

import (
	"fmt"
	"os"
)

/*
A polite exit without a panic traceback
*/
func abort(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
