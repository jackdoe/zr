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
			in: `hello world
goodbye world
new world`,
			out: []string{"hello_0", "world_0", "goodbye_1", "world_1", "new_1", "world_1"},
		},

		cases{
			in: `a b c d





e

x`,
			out: []string{"a_0", "b_0", "c_0", "d_0", "e_2", "x_3"},
		},
	} {

		r := DefaultIndexTokenizer[0].Apply([]string{ascii(c.in)})
		if !eq(c.out, r) {
			t.Fatalf("got %v, expected: %v", r, c.out)
		}
	}

}
