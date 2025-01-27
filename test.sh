#!/bin/bash

for i in {1..50000}
do
    go test -timeout 600s -count 1 -v github.com/matrixorigin/matrixcube/raftstore > test.log
    v=`tail -n 1 test.log | awk {'print $1'}`
    if [ "$v" != "ok" ]
    then
        mv ./test.log ./test-$i.log
        echo "$i: error"
    else
        echo "$i: ok"
    fi
done
