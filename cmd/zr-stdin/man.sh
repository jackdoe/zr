export MANWIDTH=80
for i in `find /usr/share/man/man[1-9] -type f -name '*.gz' | shuf`; do
    base=$(basename $i | sed -e 's/\.gz$//g')
    title=$(man -P cat $base | tr " " "\n" | head -1)
    echo $i
    man -P cat $base | zr-stdin -title "$title" -k man -id $base
done
