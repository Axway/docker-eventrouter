#!/bin/bash
#

if [ "${VERSION:-}" = ""  ]; then
    VERSION="3.0.$(date +'%Y%m%d')"
fi
sed -i "s/VERSION=.*/VERSION=$VERSION/" .env

echo VERSION=$VERSION
cat .env
