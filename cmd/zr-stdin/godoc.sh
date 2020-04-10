for i in `go list ...`; do
    echo $i
    go doc --all $i | zr-stdin -title "$i" -k godoc -id $i
done
