# ZR - offline, local, serverless search of StackOverflow
       with very low ram footprint


Check out the demo at:

  https://asciinema.org/a/t7bLU9Vfg7MSHMEMTb87GDB3v

or

  $ asciinema play https://asciinema.org/a/t7bLU9Vfg7MSHMEMTb87GDB3v

# What is it?

it is a local index of 2020 StackOverflow's Posts.xml (downloaded from
archive.org) with about 43 million questions and answers.

You can get the whole stackexchange from:

archive.org/download/stackexchange/stackexchange_archive.torrent

at the time of writing this the last post is 2020-03-01T07:17:40.850,
and whe whole stackexchange is about 65G archived, if you select only
stackovefrlow it will be 14GB archived, and when you unarchive it it
will become about 80G.

and you can use ZR for any of the stack exchange Posts.xml, though I
only tested it with SO and superuser.

# Why?

Since I went to live in the tty (https://txt.black/~jack/tty.txt), I
have been trying to reduce my dependency on the web, but it is quite
often that I have to use StackOverflow to copy pasta some code.

So here it is.

# How does it work?

I am experimenting with bloom-like searching, so the way I am building
the index is quite funky.

First the normalizer only alphanumeric (ascii) characters, and then
the tokenizer splits on whitespace and then for each token it converts
it to `gometro.Hash(token)&0x0..FFFF_token[0]`, for example "merge" will be
converted to (123123123 & 0x0..ffff)_m, so only 2 bytes of the hash
are used, and the first character of the token.

for better explanation refer to the code in data.go:

```
	first := s[0]

	h := metro.Hash64Str(s, 0)

	// 65k per starting character
	// so overall 65k * 36, or about 2.5 mil files

	return fmt.Sprintf("%x_%c", h&0x000000000000FFFF, first)
```

So this is about 2.5 million files, each containing 4 bytes per
document matching the specific term.

In the same time a weights table is created, that contains 12 bytes
per post, it holds the chosen answer id, the parent viewcount (because
only questions have viewcount) and the score (upvotes/downvotes)

This weights table is used during query time to sort the matching
posts.

I use https://github.com/rekki/go-query package's query and file index.
The way it works is at query time it just opens the term file and
loads []int32 array, then it does the query operation on it.
Lets say we have a query "git merge" this will open 2 files

root/inv/g/hash(git)%0xffff_g
root/inv/m/hash(merge)%0xffff_e

It will read the contents and create two []int32 sorted lists (they
are sorted by insertion order). You know how you can use the database
as a filesystem? You can also use the filesystem as a database :D.

Then it can efficiently merge them, so if you have a term that is
matching most of SO's questions (like 'a') it will load few million *
4 bytes in memory, well the while database is about 45 million, so
worse case you get ~200mb of data, so not the end of the world.


After that it uses binary search and other tricks to iterate fast and
do the intersection (you can check out github.com/rekki/go-query for
more details). For each match the weights table is used to extract the
"popularity" and then a score is created which is totally biased
towards viewcount.

For the topN hits we query the sqlite database, find the question
threads, sort the answers by score and pretty print them (see the
example)

Anyway, I am quite excited about this bloomy search, because you get
some very interesting properties, first the probability of getting a
mismatch decreases with the amount of tokens in the query, and so far
I have been getting only good results. I am still learning how it
behaves, but it is the first time I am exploring random in a search
problem.

There is one more trick being used, in order to prefer things on the
top of the files (threads) more than the bottom, I split the blob in
32 chunks, with chunk = max(32,sqrt(line number)), then on query time
I create a dismax query with 32 AND queries.. as I said, quite funky!

# Install

$ go get github.com/jackdoe/zr/...
$ go install github.com/jackdoe/zr/cmd/zr
$ go install github.com/jackdoe/zr/cmd/zr-stackexchange
$ go install github.com/jackdoe/zr/cmd/zr-stdin
$ go install github.com/jackdoe/zr/cmd/zr-reindex

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

You can speed it up by increasing the -batch-size factor (the bigger it
is the more ram it will use, with 1k it will use few hundred MB or
less)

also: sudo mount -o remount,noatime,nodiratime,lazytime /

The total index size is about 100G (sqlite plus inverted index).
and at least 2.5k inodes (depending on your blocksize)

# Search (example)

the query for "git merge -ubuntu -windows" is translated to

    (git AND merge) AND NOT (ubuntu OR windows)

$ ~/go/bin/zr git merge
# use zr -h to see the help

  ┌------------------------------
  │ Q: How do I force "git pull" to overwrite local files?
  │    tags:     <git><version-control><overwrite><git-pull><git-fetch>
  │    url:      stackoverflow.com/q/1125968
  │    score:    6905/4499402
  │    created:  2009-07-14T14:58:15.550
  │    accepted: stackoverflow.com/a/8888015
  │ ---
  │
  │ How do I force an overwrite of local files on a git pull ?
  │
  │ *The scenario is the following:*
  │
  │ * A team member is modifying the templates for a website we are working on
  │ * They are adding some images to the images directory (but forgets to add them
  │ under source control)
  ....

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

searching in stackoverflow, superuser and man pages by default
you can specify which index to use with `zr -k so,su,man`, by default it shows
top result from each index, but you can use -top to specify how many per index.

# index man pages using `zr-stdin`

to index your man pages run:

  for i in `find /usr/share/man/man[1-9] -type f -name '*.gz' | shuf`; do
      base=$(basename $i | sed -e 's/\.gz$//g')
      title=$(man -P cat $base | tr " " "\n" | head -1)
      echo $i

      man -P cat $base | zr-stdin -title "$title" -kind man -id $base
  done

then run `zr-reindex -k man` to build then index

# Contribute

The whole thing is free for all, so all patches are welcome
(especially if it will make your tty life better).

Especially visualization wise, I think it can be done much better.

# TODO

* sharding if needed
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

-b
