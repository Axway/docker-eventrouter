#!/bin/bash
#

set -xeuo pipefail

NAME=qlt-router
H=$HOME
#make linux-agent
I=$(realpath $(dirname "$0")/..)
. $I/.env

ls -l

rm -rf "$H/rpmbuild"
rm -rf "$H/rpmbuild/SOURCES/"*

mkdir -p "$H/rpmbuild/SPECS"
mkdir -p "$H/rpmbuild/SOURCES/$NAME-$VERSION"
mkdir -p "$H/rpmbuild/SOURCES/$NAME-1.0.0"
mkdir -p "$H/rpm"

cp -f $I/rpm/rpm.spec "$H/rpmbuild/SPECS/"
echo "$VERSION"
sed -i 's/Version:       .*/'"Version:       $VERSION"'/g' "$H/rpmbuild/SPECS/rpm.spec"
cat "$H/rpmbuild/SPECS/rpm.spec"

cp "./artefacts/$NAME" "$H/rpmbuild/SOURCES/$NAME-$VERSION/${NAME}d"
cp "$I/rpm/$NAME" "$H/rpmbuild/SOURCES/$NAME-$VERSION/$NAME"
sed -i 's/APPVERSION=.*/APPVERSION="'"$VERSION"'"/g' "$H/rpmbuild/SOURCES/$NAME-$VERSION/$NAME"
(
    cd "$H/rpmbuild/SOURCES/"
    tar cvfz "$NAME-$VERSION.tar.gz" "$NAME-$VERSION"
    cp "$NAME-$VERSION.tar.gz" "$H/rpm" #FIXME ????
    rm -rf "$NAME-$VERSION"
    find .
    find "$H/rpmbuild"
    rpmbuild -bb "$H/rpmbuild/SPECS/rpm.spec"
)

PKGROOT=$H/rpm/x86_64
if [ ! -d "$PKGROOT" ]; then
    PKGROOT=$H/rpmbuild/RPMS/x86_64
fi

ls -l "$PKGROOT"
cp "$PKGROOT"/* "$I/artefacts"
#curl -k -v -u $ARTIFACTORY_NEW_REGISTRY_USER:$ARTIFACTORY_NEW_REGISTRY_PASSWORD -T flowmanager-agent-$VERSION-1.el7.x86_64.rpm "$ARTIFACTORY_URL/$FLOWMANAGER_GENERIC_RELEASE_REPO/flowmanager-rpm-agent/axway-flowmanager-agent-$VERSION.rpm"
