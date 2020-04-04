// embedding because its not updated for years, and because its just 1 function
// better to copy paste it

/*
The MIT License (MIT)

Copyright (c) 2014 Mitchell Hashimoto

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package data

import (
	"bytes"
	"unicode"
)

func WrapString(s string, lim uint) string {
	// Initialize a buffer with a slightly larger size to account for breaks
	init := make([]byte, 0, len(s))
	buf := bytes.NewBuffer(init)
	var current uint
	var wordBuf, spaceBuf bytes.Buffer
	for _, char := range s {
		if char == '\n' {
			if wordBuf.Len() == 0 {
				if current+uint(spaceBuf.Len()) <= lim {
					_, _ = spaceBuf.WriteTo(buf)
				}
				spaceBuf.Reset()
			} else {
				_, _ = spaceBuf.WriteTo(buf)
				spaceBuf.Reset()
				_, _ = wordBuf.WriteTo(buf)
				wordBuf.Reset()
			}
			buf.WriteRune(char)
			current = 0
		} else if unicode.IsSpace(char) {
			if spaceBuf.Len() == 0 || wordBuf.Len() > 0 {
				current += uint(spaceBuf.Len() + wordBuf.Len())
				_, _ = spaceBuf.WriteTo(buf)
				spaceBuf.Reset()
				_, _ = wordBuf.WriteTo(buf)
				wordBuf.Reset()
			}
			spaceBuf.WriteRune(char)
		} else {
			wordBuf.WriteRune(char)
			if current+uint(spaceBuf.Len()+wordBuf.Len()) > lim && uint(wordBuf.Len()) < lim {
				buf.WriteRune('\n')
				current = 0
				spaceBuf.Reset()
			}
		}
	}
	if wordBuf.Len() == 0 {
		if current+uint(spaceBuf.Len()) <= lim {
			_, _ = spaceBuf.WriteTo(buf)
		}
	} else {
		_, _ = spaceBuf.WriteTo(buf)
		_, _ = wordBuf.WriteTo(buf)
	}
	return buf.String()
}
