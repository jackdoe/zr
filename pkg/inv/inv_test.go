package inv

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	iq "github.com/rekki/go-query"
	"github.com/rekki/go-query/util/index"
)

type ExampleCity struct {
	ID      int32
	Name    string
	Country string
	Names   []string
}

func (e *ExampleCity) DocumentID() int32 {
	return e.ID
}

func (e *ExampleCity) IndexableFields() map[string][]string {
	out := map[string][]string{}

	out["name"] = []string{e.Name}
	out["names"] = e.Names
	out["country"] = []string{e.Country}

	return out
}

func toDocumentsID(in []*ExampleCity) []index.DocumentWithID {
	out := make([]index.DocumentWithID, len(in))
	for i, d := range in {
		out[i] = index.DocumentWithID(d)
	}
	return out
}

func TestExampleDir(t *testing.T) {
	dir, err := ioutil.TempDir("", "forward")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	m, err := NewRocksIndex(dir, nil)
	if err != nil {
		panic(err)
	}
	list := []*ExampleCity{
		&ExampleCity{Name: "Amsterdam", Country: "NL", ID: 0},
		&ExampleCity{Name: "Amsterdam, USA", Country: "USA", ID: 1},
		&ExampleCity{Name: "London", Country: "UK", ID: 2},
		&ExampleCity{Name: "Sofia Amsterdam", Country: "BG", ID: 3},
	}

	for i := len(list); i < 10000; i++ {
		list = append(list, &ExampleCity{Name: fmt.Sprintf("London%d", i), Country: "UK", ID: int32(i)})
	}
	err = m.Index(toDocumentsID(list)...)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	q := iq.And(m.Terms("name", "aMSterdam sofia")...)

	m.Foreach(q, func(did int32, score float32) {
		city := list[did]
		log.Printf("%v matching with score %f", city, score)
		n++
	})
	if n != 1 {
		t.Fatalf("expected 1 got %d", n)
	}

	n = 0
	qq := iq.Or(m.Terms("name", "aMSterdam sofia")...)

	m.Foreach(qq, func(did int32, score float32) {
		city := list[did]
		log.Printf("%v matching with score %f", city, score)
		n++
	})
	if n != 3 {
		t.Fatalf("expected 3 got %d", n)
	}

}
