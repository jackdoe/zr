# ZR - offline, local search of StackOverflow, with very low ram
  footprint

# What is it?

it is a local index of 2020 StackOverflow's Posts.xml (downloaded from
archive.org) with about 43 million questions and answers.

You can get the whole stackexchange from:

archive.org/download/stackexchange/stackexchange_archive.torrent

at the time of writing this the last post is 2020-03-01T07:17:40.850,
and it is about 65G archived, if you select only stackovefrlow it will
be 14GB archived, and when you unarchive it it will become about 80G.

and you can use ZR for any of the stack exchange Posts.xml, though I
only tested it with SO.


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
	first := in[i][0]

	h := metro.Hash64Str(in[i], 0)

	// 65k per starting character
	// so overall 65_000 * 36, or about 2.5 mil files

	in[i] = fmt.Sprintf("%x_%c", h&0x000000000000FFFF, first)
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
root/inv/m/hash(merge)%0xffff_m

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

# Install

$ go get github.com/jackdoe/zr/...
$ go install github.com/jackdoe/zr/cmd/zr
$ go install github.com/jackdoe/zr/cmd/zr-sqlite
$ go install github.com/jackdoe/zr/cmd/zr-index

# Build the index

1. First import the xml docs into sqlite and use html2text on the body,
   converd ids to strings and tec

$ ~/go/bin/zr-sqlite -posts ~/downloads/Posts.xml
# -root is by default ~/.zr-data

This will take about 2-3 hours, it is single threaded, scan and
insert and it inserts about 5k documents per second.

You can restart it at any point, but it will start scanning from
scratch, and parsing is like 50% of the cpu time, so try to keep it
running until its done.

2. After that you need to build the inverted index and the weights
   table

$ ~/go/bin/zr-index -at-a-time 10000
# -root is by default ~/.zr-data

This is quite slower, it indexes about 3k documents per second, so it
takes like 5 hours to finish (it is easy to be sharded and etc, but I
only have 2 cores anyway, so wont be much faster me)

You can speed it up by increasing the -at-a-time factor (the bigger it
is the more ram it will use, with 1k it will use few hundred MB or less)

The total index size is about 100G (sqlite plus inverted index).
and at least 2.5k inodes (depending on your blocksize)

# Search (example)

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
│ * They are sending the images by mail, later, to me
│ * I'm adding the images under the source control and pushing them to GitHub
│ together with other changes
│ * They cannot pull updates from GitHub because Git doesn't want to overwrite
│ their files.
│
│ *This is the error I'm getting:*
│
│ >
│ >
│ >
│ > error: Untracked working tree file 'public/images/icon.gif' would be
│ > overwritten by merge
│ >
│ >
│
│ How do I force Git to overwrite them? The person is a designer - usually, I
│ resolve all the conflicts by hand, so the server has the most recent version
│ that they just need to update on their computer.
│
└--
┌-----
  A: stackoverflow.com/a/8888015 score: 9637, created: 2012-01-17T00:02:58.813
  ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

  -----------------------------------------------------------------------------
  Important: If you have any local changes, they will be lost. With or without
  --hard option, any local commits that haven't been pushed will be lost. [*]
  -----------------------------------------------------------------------------

  If you have any files that are not tracked by Git (e.g. uploaded user
  content), these files will not be affected.

  I think this is the right way:

  git fetch --all

  Then, you have two options:

  git reset --hard origin/master

  OR If you are on some other branch:

  git reset --hard origin/<branch_name>

  Explanation:
  ------------

  git fetch downloads the latest from remote without trying to merge or rebase
  anything.

  Then the git reset resets the master branch to what you just fetched. The
  --hard option changes all the files in your working tree to match the files in
  origin/master

  Maintain current local commits
  ------------------------------

  [*] : It's worth noting that it is possible to maintain current local commits
  by creating a branch from master before resetting:

  git checkout master
  git branch new-branch-to-save-current-commits
  git fetch --all
  git reset --hard origin/master

  After this, all of the old commits will be kept in
  new-branch-to-save-current-commits.

  Uncommitted changes
  -------------------

  Uncommitted changes, however (even staged), will be lost. Make sure to stash
  and commit anything you need. For that you can run the following:

  git stash

  And then to reapply these uncommitted changes:

  git stash pop

└--
┌-----
  A: stackoverflow.com/a/2798934 score: 906, created: 2010-05-09T19:45:21.437

  Try this:

  git reset --hard HEAD
  git pull

  It should do what you want.

└--
....

total: 2288, took: 23.726728ms


# Contribute

The whole thing is free for all, so all patches are welcome
(especially if it will make your tty life better).

Especially visualization wise, I think it can be done much better

# TODO

* sharding if needed
* remove stop words
* remove questions without views or without answers etc
* upload the built index somewhere so people dont have to spend day
  building it
* build the same but for godoc, man and the linux kernel docs
* add PAGER support

# What does ZR mean?

nothing, just Z and R

# Why didn't you use some other search engine

Whats the fun in that?


# Conclusion

Writing the whole project with minimal usage of the web was an amazing
experience, now 'zr git rebase faster' renders faster than 'man
git-rebase'

-b

