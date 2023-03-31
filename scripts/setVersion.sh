#!/bin/bash
#

if [ "${VERSION:-}" = ""  ]; then
    VERSION="1.0.$(date +'%Y%m%d')"    
fi
sed -i "s/VERSION=.*/VERSION=$VERSION/" .env

echo VERSION=$VERSION
cat .env
