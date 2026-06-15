#!/bin/bash
for b in $(git branch --format='%(refname:short)' | grep '^chore/'); do
    changes=$(git diff --shortstat main.."$b" 2>/dev/null | sed 's/^ *//')
    if [ -n "$changes" ]; then
        echo "$changes | $b"
    fi
done
