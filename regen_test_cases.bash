#!/usr/bin/env bash

for ii in $(find fixtures -name "*.go")
do
    goblin -file $ii | json_pp > $(dirname $ii)/$(basename $ii .go).json
done

for ii in $(find fixtures -name "*.go.txt")
do
    goblin -expr "$(cat $ii)" | json_pp > $(dirname $ii)/$(basename $ii .go.txt).json
done
