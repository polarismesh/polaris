#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

workdir="$(dirname "$0")"
. "$workdir/export.sh"

for dir in "${DIRS[@]}" ; do
    v3dir="$dir/v3"
    modified=$(git status --porcelain "$v3dir")
    if [[ -n "${modified}" ]]; then
        printf "\nerror: Make sure to not edit the auto generated files in %s \n" "$v3dir"
        exit 1
    fi
done