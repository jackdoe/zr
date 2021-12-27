# ZR - offline, local, serverless search of StackOverflow
       with very low ram footprint

Check out the demo at:

  https://asciinema.org/a/Gban33qB188dK5P4yBcooubEw 

or

  $ asciinema play https://asciinema.org/a/Gban33qB188dK5P4yBcooubEw 

# TLDR install

----

Replace .bin with your local bin directory, and replace linux with
darwin (for macos) or windows.

# 1. download zr
mkdir -p ~/.bin \
   && curl -sL https://github.com/jackdoe/zr/releases/download/v0.0.17/zr_v0.0.17_linux_amd64.tar.gz \
        | tar --exclude README.txt --exclude LICENSE -C ~/.bin -xzvf -

# 2. download man and rfc indexes
~/.bin/zr-fetch

# 3. set your preferred query order
echo '{"kind": ["public/man","public/rfc"]}' > ~/.zr/config.json

# 4. enjoy
~/.bin/zr transmission control protocol
~/.bin/zr printf

----

# CHANGELOG - 04/2019

  * now you can index man pages as well (or any kinds of documents)
  * scoring is much better by using some tricks to index the
    sqrt of the line numbers (read details in 'How does it work?'
  * added support for NOT queries
      'reset password -mysql -amazon'
    will produce:
      (reset AND password) AND NOT (mysql OR amazon)
  * pager support, tries to use less, more using:
    https://github.com/jackdoe/go-pager
  * multi index search
  * use sqlite3 to store the inverted index and remove the bloom search
    (bloom search was very interesting, but it needs at least two
    terms to make sense, which is super annoying)
  * add sharding
  * zr-fetch - a way to download public indexes

# What is it?

it is a local index of 2020 StackOverflow's Posts.xml (downloaded from
archive.org) with about 47 million questions and answers.

You can get the whole stackexchange from:

archive.org/download/stackexchange/stackexchange_archive.torrent

at the time of writing this the last post is 2020-03-01T07:17:40.850,
and whe whole stackexchange is about 65G archived, if you select only
stackovefrlow it will be 14GB archived, and when you unarchive it it
will become about 80G.

and you can use ZR for any of the stack exchange Posts.xml, though I
only tested it with SO and superuser.

# Why?

Since I went to live in the tty (https://punkjazz.org/jack/tty.txt), I
have been trying to reduce my dependency on the web, but it is quite
often that I have to use StackOverflow to copy pasta some code.

So here it is.

# How does it work?

I am experimenting with somewhat weird search index.

First the normalizer only alphanumeric (ascii) characters, and then
the tokenizer splits on whitespace and then for each token it adds a
sufix with int(max(16, sqrt(line_number)). So if we have the following
document:

  hello world
  goodbye world
  new world

it will create the tokens:

  hello_0 world_0 goodbye_1 world_1 new_1 world_1

Then at query time it will build a bunch of dismax queries with custom
weights based on _0, _1 .. etc so _0 has weight 16, _1 has 15.. etc,
this way the top of the file has more weight than the bottom, but also
all queries are somewhat phrase queries, the lower in the file it
gets, the more lose the phrase is.

I use https://github.com/rekki/go-query package's query and
https://github.com/jackdoe/go-query-sql to store the indexes in
sqlite.

Its a simple table: (id varchar(255), list largeblob) and the blob is
a sorted list of 4 byte integers.

If you have a term that is matching most of SO's questions (like 'a')
it will load few million * 4 bytes in memory, well the while database
is about 45 million, so worse case you get ~200mb of data, so not the
end of the world.

BTW because of the line number suffix, the distribution of the words
is much better, as in you dont have a word that exists in every single
document.

After that it uses binary search and other tricks to iterate fast and
do the intersection (you can check out github.com/rekki/go-query for
more details). For each match the weights table is used to extract the
"popularity" and then a score is created which is totally biased
towards viewcount.

For the topN hits we query the sqlite database, find the documents,
sort the answers by score and pretty print them (see the example)

# Size

The documents are compressed with snappy, for 80GB stackoverflow
posts, the snappy compressed sqlite is about 35GB, and the inverted
index is about 40GB.

# Install

$ go get github.com/jackdoe/zr/...
$ go install github.com/jackdoe/zr/cmd/zr
$ go install github.com/jackdoe/zr/cmd/zr-stackexchange
$ go install github.com/jackdoe/zr/cmd/zr-stdin
$ go install github.com/jackdoe/zr/cmd/zr-reindex
$ go install github.com/jackdoe/zr/cmd/zr-fetch

or you can download the binary from releases/

# Download the public index
You can download and use the public index I build and publish, it
includes man pages and RFC

$ zr-fetch

This is the equivalent of:
cat << EOF | zr-fetch -list -
public/man https://punkjazz.org/jack/zr-public/man.tar.gz
public/rfc https://punkjazz.org/jack/zr-public/rfc.tar.gz
EOF

By default it reads https://raw.githubusercontent.com/jackdoe/zr/master/public.txt
but you can specify any url or - for stdin

$ zr -k public/man printf
$ zr -k public/rfc transmission control protocol

# Build the index

1. First import the xml docs into sqlite and use html2text on the body,
   converd ids to strings and tec

$ cat Posts.xml \
      | sort --stable -S 1G -t '"' -k 4 --numeric-sort \
      | tail -n +4 \
      | zr-stackexchange -k so -url-base stackoverflow.com

# sort the post by questions first and ignore </posts that will be on
# top
# -root is by default ~/.zr-data
# -k means kind,
#  I use so for stackoverflow, su for superuser and man for man

This will take about 2-3 hours, it is single threaded, scan and
insert and it inserts about 5k documents per second.

You can restart it at any point, but it will start scanning from
scratch, and parsing is like 50% of the cpu time, so try to keep it
running until its done.

2. After that you need to build the inverted index

$ ~/go/bin/zr-reindex -k so

This is quite slower, it indexes about 3k documents per second, so it
takes like 5 hours to finish (it is easy to be sharded and etc, but I
only have 2 cores anyway, so wont be much faster me)

You can speed it up by increasing the -batch factor (the bigger it
is the more ram it will use, with 1k it will use few hundred MB or
less)

also: sudo mount -o remount,noatime,nodiratime,lazytime /

The total index size is about 30G (sqlite plus inverted index).
(the documents are compressed with snappy)

# Search (example)

the query for "git merge -ubuntu -windows" is translated to

    (git AND merge) AND NOT (ubuntu OR windows)

$ ~/go/bin/zr git merge
# use zr -h to see the help

  ██████████████████████████████████████ so ██████████████████████████████████████
  
  ┌------------------------------
  │ Q: Is there a "theirs" version of "git merge -s ours"?
  │    tags:     git,git-merge
  │    url:      stackoverflow.com/q/173919
  │    score:    850/445471
  │    created:  2008-10-06T11:16:43.217
  │    accepted: stackoverflow.com/a/3364506
  │ ---
  │ 
  │ When merging topic branch "B" into "A" using git merge , I get some conflicts.
  │ I know all the conflicts can be solved using the version in "B".
  │ 
  │ I am aware of git merge -s ours. But what I want is something like git merge
  │ -s theirs.
  │ 
  │ Why doesn't it exist? How can I achieve the same result after the conflicting
  │ merge with existing git commands? ( git checkout every unmerged file from B)
  │ 
  │ UPDATE: The "solution" of just discarding anything from branch A (the merge
  │ commit point to B version of the tree) is not what I am looking for.
  │ 
  └--
  
  ┌-----
    A: stackoverflow.com/a/173954 score: 21, created: 2008-10-06T11:34:59.203
    
    I solved my problem using
    
    git checkout -m old
    git checkout -b new B
    git merge -s ours old
    ...

  █████████████████████████████████████ man █████████████████████████████████████

  GIT-MERGE-BASE(1)                 Git Manual                 GIT-MERGE-BASE(1)

  NAME
         git-merge-base - Find as good common ancestors as possible for a merge

  SYNOPSIS
         git merge-base [-a|--all] <commit> <commit>...
         git merge-base [-a|--all] --octopus <commit>...
         git merge-base --is-ancestor <commit> <commit>
         git merge-base --independent <commit>...
         git merge-base --fork-point <ref> [<commit>]

  DESCRIPTION
         git merge-base finds best common ancestor(s) between two commits to use
         in a three-way merge. One common ancestor is better than another common
         ancestor if the latter is an ancestor of the former. A common ancestor
         that does not have any better common ancestor is a best common
         ancestor, i.e. a merge base. Note that there can be more than one merge
         base for a pair of commits.
  ...

  ██████████████████████████████████████ su ██████████████████████████████████████
  
  ┌------------------------------
  │ Q: Is there a way to redo a merge in git?
  │    tags:     git
  │    url:      superuser.com/q/691494
  │    score:    17/17366
  │    created:  2013-12-21T07:01:22.967
  │ ---
  │ 
  │ So I made a pretty big mistake. I made a commit, pulled, merged (but messed up
  │ the code while doing so) and then pushed. I'd like to redo that merge and get
  │ the code right. Is there any way to do this?
  │ 
  │ I use bitbucket.
  │ 
  └--
  
  ┌-----
    A: superuser.com/a/691495 score: 1, created: 2013-12-21T07:03:44.313
    
    git merge --abort and then you can merge again
    
  └--
  ...

searching in stackoverflow, superuser and man pages by default
you can specify which index to use with `zr -k so,su,man`, by default it shows
top result from each index, but you can use -top to specify how many per index.

# index man pages using `zr-stdin`

pagerank the manpages by building a graph of the references.

example script:

  # install ripgrep and github.com/jackdoe/updown/cmd/pagerank
  rg -z '.BR \w+ \(' /usr/share/man/man[1-9]/* \
      | sed -e 's/ //g' \
      | sed -e 's/.BR//g' \
      | tr '(' '.' \
      | cut -f 1 -d ')' \
      | sed -e 's/.*\///g' \
      | sed -e 's/\.gz:/ /' \
      | pagerank -int -prob-follow 0.6 -tolerance 0.001 > /tmp/zr-man-pagerank
  
  	  
  for i in `find /usr/share/man/man[1-9] -type f -name '*.gz' | shuf`; do
      base=$(basename $i | sed -e 's/\.gz//g')
      title=$(man -P cat $base | tr " " "\n" | head -1)
      score=$(cat /tmp/zr-man-pagerank | grep -w $base | cut -f 1 -d ' ')
      popularity=${score:-0}
      echo $base score: $popularity
      man -P cat $base | zr-stdin -title "$title" -k man -id $base -popularity $popularity
  done


then run `zr-reindex -k man` to build then index

# index go doc

I run this script that generates godoc for all installed packages, and
also sets the popularity to be the pagerank of how they are imported
from each other

  # need to go install github.com/jackdoe/updown/cmd/pagerank
  # need to go install github.com/jackdoe/updown/cmd/printimports
  
  find $GOPATH/src -type f -name '*.go' -exec printimports -file {} \; \
      | pagerank -int -tolerance 0.01 -prob-follow 0.65 \
      | sort -n > /tmp/zr-go-pagerank 
  
  for x in `cat /tmp/zr-go-pagerank | sed -e 's/ /:/g'`; do
      score=$(echo $x | cut -f 1 -d ':')
      package=$(echo $x | cut -f 2 -d ':')
      echo $package - score "$score"
  
      go doc --all $package | zr-stdin -title "$package" -k godoc -id $package -popularity "$score"
  done


# Contribute

The whole thing is free for all, so all patches are welcome
(especially if it will make your tty life better).

Especially visualization wise, I think it can be done much better.

# TODO

* remove stop words
* upload the built index somewhere so people dont have to spend day
  building it

# What does ZR mean?

Nothing, just Z and R.

# Why didn't you use some other search engine?

Whats the fun in that?

I tried few lsmt based stores (badger, rocksdb) but the cost of openning
and closing the db is too high for command line app, so I went back to
file based postings lists.

Also tried roaring bitmaps to store the lists, but because so many are
with close to zero documents the overhead is too high.

Tried meilisearch as well, but it consumes a lot of disk and ram and
it is slow to index on my laptop (8g ram).

# Conclusion

Writing the whole project with minimal usage of the web was an amazing
experience, now 'zr git rebase master' renders faster than 'man
git-rebase'.

At some point I got stuck into adding sharding to maximize resource
usage (and I did), but then I just thought: what about if I just chill
and let it run over night, so it is done in 8 hours instead of 2-3?
its not the end of the world.
(NOTE: later I actually added sharding haha)

Also DAMN sqlite3 is some good piece of software!

-b
