package util

import "github.com/golang/snappy"

func Compress(b []byte) []byte {
	return snappy.Encode(nil, b)
}

func CompressX(a ...[]byte) []byte {
	combined := []byte{}
	for _, b := range a {
		combined = append(combined, b...)
	}
	return snappy.Encode(nil, combined)
}

func JoinB(a ...[]byte) []byte {
	combined := []byte{}
	for _, b := range a {
		combined = append(combined, b...)
	}
	return combined
}

func Decompress(b []byte) []byte {
	out, err := snappy.Decode(nil, b)
	if err != nil {
		panic(err)
	}
	return out
}
