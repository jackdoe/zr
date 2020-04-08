package data

import (
	"fmt"
	"testing"
)

func TestTokenize(t *testing.T) {
	for _, e := range []string{
		`line 0
line 1
line 2
line           3 a b c d
line 5
line 6
line 7
line 8
line 9
`, "", " ", "a b c"} {

		fmt.Printf("\n%v\n", DefaultIndexTokenizer[0].Apply([]string{e}))

		fmt.Printf("\nNOEM: %v\n", ascii.Apply(e))
	}

}
