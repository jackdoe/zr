package data

import (
	"math"
	"strconv"
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
	first := s[0]
	h := metro.Hash64Str(s, 0)

	// 65k per starting character
	// so overall 65k * 36, or about 2.5 mil files

	var sb strings.Builder
	sb.WriteString(strconv.FormatUint(h&0x000000000000FFFF, 10))
	sb.WriteRune('_')
	sb.WriteRune(rune(first))
	return sb.String()
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

const MAX_CHUNKS = 16

// haha this is extreme hack
var DefaultIndexTokenizer = []tokenize.Tokenizer{
	tokenize.NewCustom(func(in []string) []string {
		out := make([]string, 0, len(in))

		var sb strings.Builder

		lineNo := 0
		chunkID := uint64(0)
		sb.WriteString(strconv.FormatUint(chunkID, 10))
		sb.WriteRune('_')
		for _, current := range in {
			if len(current) == 0 {
				continue
			}
			hasChar := false
			for _, c := range current {
				if c == '\n' || c == '\r' {
					lineNo++
					chunkID = uint64(math.Sqrt(float64(lineNo + 1)))
					if chunkID > MAX_CHUNKS {
						chunkID = MAX_CHUNKS
					}
				}

				if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
					if sb.Len() > 0 && hasChar {
						if sb.Len() > 0 {
							sb.WriteRune('_')
							sb.WriteString(strconv.FormatUint(chunkID, 10))

							out = append(out, sb.String())
						}
						sb.Reset()

						hasChar = false
					}
				} else {
					hasChar = true
					sb.WriteRune(c)
				}
			}
			if sb.Len() > 0 && hasChar {
				sb.WriteRune('_')
				sb.WriteString(strconv.FormatUint(chunkID, 10))

				out = append(out, sb.String())
				sb.Reset()
			}
		}
		return out
	}),
	trimmer,
	tokenize.NewUnique(),
}

var DefaultAnalyzer = analyzer.NewAnalyzer(DefaultNormalizer, DefaultSearchTokenizer, DefaultIndexTokenizer)
