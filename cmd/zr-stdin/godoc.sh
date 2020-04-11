#!/bin/bash

# need to go install github.com/jackdoe/updown/cmd/pagerank
# need to go install github.com/jackdoe/updown/cmd/printimports

# try some basic pagerank, math fmt and testing seems to be top
# ...
# 5476 os
# 5551 google.golang.org/grpc
# 6210 bytes
# 7070 io
# 7894 time
# 8866 strings
# 10131 context
# 10775 github.com/gogo/protobuf/proto
# 15036 github.com/golang/protobuf/proto
# 18180 testing
# 22390 math
# 36642 fmt

find $GOPATH/src -type f -name '*.go' -exec printimports -file {} \; \
    | pagerank -int -tolerance 0.01 -prob-follow 0.65 \
    | sort -n > /tmp/zr-go-pagerank 

for x in `cat /tmp/zr-go-pagerank | sed -e 's/ /:/g'`; do
    score=$(echo $x | cut -f 1 -d ':')
    package=$(echo $x | cut -f 2 -d ':')
    echo $package - score "$score"

    ( go doc --all $package | zr-stdin -title "$package" -k godoc -id $package -popularity "$score" ) &
done
