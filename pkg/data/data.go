package data

import (
	"math"
	"strconv"
	"strings"

	analyzer "github.com/rekki/go-query-analyze"
	norm "github.com/rekki/go-query-analyze/normalize"
	"github.com/rekki/go-query-analyze/tokenize"
)

func ascii(s string) string {
	var sb strings.Builder
	hadSpace := false
	for _, c := range s {
		if c == '\n' || c == '\r' {
			sb.WriteRune('\n')
			continue
		}
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
				sb.WriteRune(' ')
				hadSpace = true
			}
		}
	}
	return sb.String()
}

var MAX_TOKEN_SIZE = 10

func prefixLine(in []tokenize.Token) []tokenize.Token {
	out := make([]tokenize.Token, 0, len(in))

	var sb strings.Builder

	lineNo := 0
	chunkID := uint64(0)
	for _, current := range in {
		if len(current.Text) == 0 {
			continue
		}
		hasChar := false
		for _, c := range current.Text {
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
				if sb.Len() > 0 && hasChar {
					if sb.Len() > 0 {
						sb.WriteRune('_')
						sb.WriteString(strconv.FormatUint(chunkID, 10))

						out = append(out, current.Clone(sb.String()))
					}
					sb.Reset()

					hasChar = false
				}
			} else {
				hasChar = true
				if sb.Len() < MAX_TOKEN_SIZE {
					sb.WriteRune(c)
				}
			}
			if c == '\n' || c == '\r' {
				lineNo++
				chunkID = uint64(math.Sqrt(float64(lineNo + 1)))
				if chunkID > MAX_CHUNKS {
					chunkID = MAX_CHUNKS
				}
			}
		}
		if sb.Len() > 0 && hasChar {
			sb.WriteRune('_')
			sb.WriteString(strconv.FormatUint(chunkID, 10))

			out = append(out, current.Clone(sb.String()))
			sb.Reset()
		}
	}
	return out
}

var DefaultNormalizer = []norm.Normalizer{norm.NewCustom(ascii)}

var DefaultSearchTokenizer = []tokenize.Tokenizer{
	tokenize.NewWhitespace(),
}

const MAX_CHUNKS = 32

// haha this is extreme hack
var DefaultIndexTokenizer = []tokenize.Tokenizer{
	tokenize.NewCustom(prefixLine),
	tokenize.NewUnique(),
}

var DefaultAnalyzer = analyzer.NewAnalyzer(DefaultNormalizer, DefaultSearchTokenizer, DefaultIndexTokenizer)
