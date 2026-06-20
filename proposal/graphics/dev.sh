#!/bin/bash

# = Development skript to build the typst documents.
# Ensure you have tpyst installed.
# Note: the log might be a little bugged, in that case just copy the command itself.

# Kill all child processes of this script upon exit
trap 'kill $(jobs -p) 2>/dev/null' EXIT

(typst watch renderer.typ --format svg | sed "s/^/[Structure Contract-LIB] /") &
(typst watch renderer.typ --format pdf | sed "s/^/[Structure Contract-LIB] /") &

#(./task2.sh | sed "s/^/[Task 2] /") &

wait
