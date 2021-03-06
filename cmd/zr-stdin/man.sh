# the score of each man page is how many times it was referenced
# by other man pages

# find all 'BOLD word (' e.g. '.BR printf (1' is printf(1), so we
# count it as reference, the we just greop in /tmp/zr_man_index how
# many times it was referenced and use it as boost


# transform:
#   /usr/share/man/man1/autoreconf.1.gz:.BR autoconf (1),
# to
#   autoreconf.1 autoconf.1

# install ripgrep and go install github.com/jackdoe/updown/cmd/pagerank

rg -z '.BR \w+ \(' /usr/share/man/man[1-9]/* \
    | sed -e 's/ //g' \
    | sed -e 's/.BR//g' \
    | tr '(' '.' \
    | cut -f 1 -d ')' \
    | sed -e 's/.*\///g' \
    | sed -e 's/\.gz:/ /' \
    | pagerank -int -prob-follow 0.6 -tolerance 0.001 > /tmp/zr-man-pagerank

	  
export MANWIDTH=80

for i in `find /usr/share/man/man[1-9] -type f -name '*.gz' | shuf`; do
    base=$(basename $i | sed -e 's/\.gz//g')
    title=$(man -P cat $base | tr " " "\n" | head -1)
    score=$(cat /tmp/zr-man-pagerank | grep -w $base | cut -f 1 -d ' ')
    popularity=${score:-0}
    echo $base score: $popularity
    ( man -P cat $base | zr-stdin -title "$title" -k man -id $base -popularity $popularity) &
done

for job in `jobs -p`
do
    echo $job
    wait $job
done
