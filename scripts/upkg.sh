#!/bin/bash
#

set -euo pipefail

folder="./src/main/ui/upkg"

if [ ! -d "$folder" ]; then
    echo "$folder does not exist"
    exit 1
fi

#grep https://unpkg.com ./src/main/ui/index.html | sed 's/.*<script src="\(.*\)".*/\1/g' 
cat ./upkg.conf | while read -r u; do
    file=$(basename "$u" )
    echo "fetching $u... $file"
    rm -f "$folder/$file"
    curl -L "$u" --output "$folder/$file"
done

