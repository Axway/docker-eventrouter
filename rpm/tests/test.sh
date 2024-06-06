#!/bin/bash
#

set -euo pipefail

mode=$2 # root|non-root
NAME=$1

I=$(realpath "$(dirname "$(realpath "$0")")")
. "$I/test-common.sh"

CONFIG="$I/$NAME"

cd /tmp

export HOME=/tmp/user
mkdir -p $HOME

case $mode in
root)
    check_ok cat /usr/bin/$NAME
;;
non-root)
    INSTALL=/tmp/user/$NAME-install
    rpm -ivh --dbpath $INSTALL/rpmdb --prefix $INSTALL /*rpm
    export PATH=$PATH:$INSTALL/usr/bin
    find $INSTALL
    check_ok cat $INSTALL/usr/bin/$NAME
;;
*)
    echo "Usage: $0 <name> <root|non-root>"
    exit 1
esac



RUNTIME=/tmp/user/$NAME-runtime

check_ok $NAME help
check_ok $NAME init "$RUNTIME"
check_ok cat $RUNTIME/etc/$NAME.conf
check_ok diff $RUNTIME/etc/$NAME.conf $CONFIG/$NAME-origin.conf

cp -r $CONFIG/* $RUNTIME/etc 
check_ok cat $RUNTIME/etc/$NAME.conf

check_ok $NAME --runtime "$RUNTIME" runtime
check_ok $NAME --runtime "$RUNTIME" --version

check_ko $NAME --runtime "$RUNTIME" stop
check_ok $NAME --runtime "$RUNTIME" start
sleep 1
check_ok $NAME --runtime "$RUNTIME" logs
check_ok $NAME --runtime "$RUNTIME" live
check_ok $NAME --runtime "$RUNTIME" ready
check_ok $NAME --runtime "$RUNTIME" metrics
check_ok $NAME --runtime "$RUNTIME" status
check_ko $NAME --runtime "$RUNTIME" start
check_ok $NAME --runtime "$RUNTIME" logs
check_ok $NAME --runtime "$RUNTIME" support
check_ok $NAME --runtime "$RUNTIME" stop
check_ok $NAME --runtime "$RUNTIME" logs
check_ko $NAME --runtime "$RUNTIME" status
check_ko $NAME --runtime "$RUNTIME" support
rm -rf /tmp/$NAME-support
check_ok $NAME --runtime "$RUNTIME" support
check_ok tar tvfz /tmp/$NAME-support.tar.gz
echo_test "Support file output"
find /tmp/$NAME-support -type f | while read file; do
    echo "====$file===="
    cat "$file"
    echo ""
done
check_ok $NAME --runtime "$RUNTIME" gen-systemd-unit
