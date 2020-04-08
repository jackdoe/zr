package data

import (
	"fmt"
	"math"
	"strings"

	"github.com/dgryski/go-metro"
	"github.com/rekki/go-query/util/analyzer"
	"github.com/rekki/go-query/util/norm"
	"github.com/rekki/go-query/util/tokenize"
)

var ascii = norm.NewCustom(func(s string) string {
	var sb strings.Builder
	hadSpace := false
	for _, c := range s {
		if ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || ('0' <= c && c <= '9') {
			if 'A' <= c && c <= 'Z' {
				c += 32
			}
			sb.WriteRune(c)
			hadSpace = false
		} else {
			if hadSpace {
				continue
			} else {
				if c == '\n' {
					sb.WriteRune('\n')
				} else {
					sb.WriteRune(' ')
				}
				hadSpace = true
			}
		}
	}
	return sb.String()
})

var DefaultNormalizer = []norm.Normalizer{ascii}

var trim = func(s string) string {
	middle := s[len(s)/2]

	h := metro.Hash64Str(s, 0)

	// 65k per starting character
	// so overall 65k * 36, or about 2.5 mil files

	return fmt.Sprintf("%x_%c", h&0x000000000000FFFF, middle)

}

var trimmer = tokenize.NewCustom(func(in []string) []string {
	for i := range in {
		in[i] = trim(in[i])
	}
	return in
})

var DefaultSearchTokenizer = []tokenize.Tokenizer{
	tokenize.NewWhitespace(),
	trimmer,
}

const MAX_CHUNKS = 32

// haha this is extreme hack
var DefaultIndexTokenizer = []tokenize.Tokenizer{
	tokenize.NewCustom(func(in []string) []string {
		out := make([]string, 0, len(in))

		var sb strings.Builder

		lineNo := 0
		chunkID := 0
		chunk := fmt.Sprintf("%d_", chunkID)
		sb.WriteString(chunk)

		for _, current := range in {
			hasChar := true
			if len(current) == 0 {
				continue
			}
			for _, c := range current {
				if c == '\n' || c == '\r' {
					lineNo++
					chunkID = int(math.Sqrt(float64(lineNo + 1)))
					if chunkID > MAX_CHUNKS {
						chunkID = MAX_CHUNKS
					}
					chunk = fmt.Sprintf("%d_", chunkID)
				}

				if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
					if sb.Len() > 0 && hasChar {
						if sb.Len() > len(chunk) {
							out = append(out, sb.String())
						}
						sb.Reset()
						sb.WriteString(chunk)
						hasChar = false
					}
				} else {
					hasChar = true
					sb.WriteRune(c)
				}

			}
			if sb.Len() > len(chunk) {
				out = append(out, sb.String())
				sb.Reset()
				sb.WriteString(chunk)
			}
		}
		return out
	}),
	trimmer,
	tokenize.NewUnique(),
}

var DefaultAnalyzer = analyzer.NewAnalyzer(DefaultNormalizer, DefaultSearchTokenizer, DefaultIndexTokenizer)
