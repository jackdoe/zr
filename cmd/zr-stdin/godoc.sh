#!/bin/bash
if [ "x$GITHUB_OAUTH_APP" = "x" ]; then
    echo 'make oauth app and set GITHUB_OAUTH_APP to client_id:client_secret this way you get 5000 api calls per hour'
    exit 1
fi

for i in `go list ... | grep -v internal | grep -v ^cmd/`; do
    score=0
    echo $i | grep 'github.com' >/dev/null 2>&1
    if [ $? = 0 ]; then
	user=$(echo $i | cut -f 2 -d '/')
	repo=$(echo $i | cut -f 3 -d '/')

	if [ "$i" = "github.com/$user/$repo" ]; then
	    score=$(curl -u $GITHUB_OAUTH_APP -s https://api.github.com/repos/$user/$repo | jq -r '.stargazers_count // 0')
	    sleep 1
	fi
    fi
    echo $i - score "$score"
    go doc --all $i | zr-stdin -title "$i" -k godoc -id $i -popularity "$score"
done
