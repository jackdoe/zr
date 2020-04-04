package util

func Chunked(chunkSize int, sliceLen int, cb func(from, to int)) {
	for i := 0; i < sliceLen; i += chunkSize {
		end := i + chunkSize

		if end > sliceLen {
			end = sliceLen
		}
		cb(i, end)
	}
}
