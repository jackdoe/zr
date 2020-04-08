package util

import "github.com/golang/snappy"

func Compress(b []byte) []byte {
	return snappy.Encode(nil, b)
}

func Decompress(b []byte) []byte {
	out, err := snappy.Decode(nil, b)
	if err != nil {
		panic(err)
	}
	return out
}
