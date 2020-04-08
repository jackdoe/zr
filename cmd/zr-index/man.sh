for i in `find /usr/share/man/man[1-9] -type f -name '*.gz' | shuf`; do
    base=$(basename $i | sed -e 's/\.gz$//g')
    title=$(man -P cat $base | tr " " "\n" | head -1)
    echo $i
    man -P cat $base | ./zr-index -title "$title" -kind man -id $base
done
