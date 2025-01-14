#!/bin/bash
#

DATE="$(date +'%Y%m%d')"
sed -i -r "s/VERSION=(.*)dev/VERSION=\1.$DATE/" .env

echo VERSION=$VERSION
cat .env
