package data

import (
	"testing"
)

func eq(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, x := range a {
		if x != b[i] {
			return false
		}
	}
	return true
}
func TestTokenize(t *testing.T) {
	type cases struct {
		in  string
		out []string
	}

	for _, c := range []cases{
		cases{
			in: `a b c d
e
`,
			out: []string{"a_0", "b_0", "c_0", "d_0", "e_1"},
		},

		cases{
			in: `a b c d





e

x`,
			out: []string{"a_0", "b_0", "c_0", "d_0", "e_2", "x_3"},
		},
		cases{
			in: `
PASTE(1)                                                                               User Commands                                                                               PASTE(1)

NAME
       paste - merge lines of files

SYNOPSIS
       paste [OPTION]... [FILE]...

DESCRIPTION
       Write lines consisting of the sequentially corresponding lines from each FILE, separated by TABs, to standard output.
`,
			out: []string{"paste_1", "1_1", "user_1", "commands_1", "paste_1", "1_1", "name_2", "paste_2", "merge_2", "lines_2", "of_2", "files_2", "synopsis_2", "paste_2", "option_2", "file_2", "description_3", "write_3", "lines_3", "consisting_3", "of_3", "the_3", "sequentially_3", "corresponding_3", "lines_3", "from_3", "each_3", "file_3", "separated_3", "by_3", "tabs_3", "to_3", "standard_3", "output_3"},
		},
	} {

		r := DefaultIndexTokenizer[0].Apply([]string{ascii(c.in)})
		if !eq(c.out, r) {
			t.Fatalf("got %v, expected: %v", r, c.out)
		}
	}

}
