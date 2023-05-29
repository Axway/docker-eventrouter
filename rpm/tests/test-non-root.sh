#!/bin/bash
#

set -euo pipefail

NAME=qlt-router

# Reset
ColorOff='\033[0m'       # Text Reset

# Regular Colors
Black='\033[0;30m'        # Black
Red='\033[0;31m'          # Red
Green='\033[0;32m'        # Green
Yellow='\033[0;33m'       # Yellow
Blue='\033[0;34m'         # Blue
Purple='\033[0;35m'       # Purple
Cyan='\033[0;36m'         # Cyan
White='\033[0;37m'        # White

# Bold
BBlack='\033[1;30m'       # Black
BRed='\033[1;31m'         # Red
BGreen='\033[1;32m'       # Green
BYellow='\033[1;33m'      # Yellow
BBlue='\033[1;34m'        # Blue
BPurple='\033[1;35m'      # Purple
BCyan='\033[1;36m'        # Cyan
BWhite='\033[1;37m'       # White

# Underline
UBlack='\033[4;30m'       # Black
URed='\033[4;31m'         # Red
UGreen='\033[4;32m'       # Green
UYellow='\033[4;33m'      # Yellow
UBlue='\033[4;34m'        # Blue
UPurple='\033[4;35m'      # Purple
UCyan='\033[4;36m'        # Cyan
UWhite='\033[4;37m'       # White

# Background
On_Black='\033[40m'       # Black
On_Red='\033[41m'         # Red
On_Green='\033[42m'       # Green
On_Yellow='\033[43m'      # Yellow
On_Blue='\033[44m'        # Blue
On_Purple='\033[45m'      # Purple
On_Cyan='\033[46m'        # Cyan
On_White='\033[47m'       # White

# High Intensity
IBlack='\033[0;90m'       # Black
IRed='\033[0;91m'         # Red
IGreen='\033[0;92m'       # Green
IYellow='\033[0;93m'      # Yellow
IBlue='\033[0;94m'        # Blue
IPurple='\033[0;95m'      # Purple
ICyan='\033[0;96m'        # Cyan
IWhite='\033[0;97m'       # White

# Bold High Intensity
BIBlack='\033[1;90m'      # Black
BIRed='\033[1;91m'        # Red
BIGreen='\033[1;92m'      # Green
BIYellow='\033[1;93m'     # Yellow
BIBlue='\033[1;94m'       # Blue
BIPurple='\033[1;95m'     # Purple
BICyan='\033[1;96m'       # Cyan
BIWhite='\033[1;97m'      # White

# High Intensity backgrounds
On_IBlack='\033[0;100m'   # Black
On_IRed='\033[0;101m'     # Red
On_IGreen='\033[0;102m'   # Green
On_IYellow='\033[0;103m'  # Yellow
On_IBlue='\033[0;104m'    # Blue
On_IPurple='\033[0;105m'  # Purple
On_ICyan='\033[0;106m'    # Cyan
On_IWhite='\033[0;107m'   # White

prefix="[$(basename "$0")]"

trap 'errormsg $LINENO -- ${BASH_LINENO[*]}' ERR

errormsg() {
    error "$0: unexpected error status:$? -- lines:" "$@"
    exit 1
}


check_ko() {
    echo_test KO "$@"
    if "$@"; then
        error "$* should fail"
        exit 1
    fi
}

check_ok() {
    echo_test OK "$@"
    if ! "$@"; then
        error "$* should not fail"
        exit 1
    fi
}

dt() {
    local d1
    d1="$(date -Ins --utc)"
    local d2="${d1//+00:00/Z}"
    local d3="${d2//,/.}"
    echo "$d3"
}

error() {
  echo -e "$Red$(dt) $prefix ERR $* $ColorOff" >&2
}

warn() {
  echo -e "$Purple$(dt) $prefix WRN $* $ColorOff" >&2
}

echo_test() {
    echo -e "$Purple$(dt) $prefix DBG $* $ColorOff" >&2
}

cd /tmp

export HOME=/tmp/user
mkdir -p $HOME

INSTALL=/tmp/user/$NAME-install
rpm -ivh --dbpath $INSTALL/rpmdb --prefix $INSTALL /*rpm
export PATH=$PATH:$INSTALL/usr/bin

find $INSTALL
RUNTIME=/tmp/user/$NAME-runtime

check_ok $NAME help
check_ok $NAME init "$RUNTIME"
check_ok cat $RUNTIME/etc/$NAME.conf

cat >$RUNTIME/etc/$NAME.conf <<EOF
accept-eula=true
name=zouzou
dosa=/config/tests/env-axway/dosa.json
dosa-key=/config/tests/env-axway/dosa-key.pem
EOF

check_ok cat $RUNTIME/etc/$NAME.conf

check_ok $NAME --runtime "$RUNTIME" runtime
check_ok $NAME --runtime "$RUNTIME" --version
#$NAME --runtime "$RUNTIME" config

check_ko $NAME --runtime "$RUNTIME" stop
check_ok $NAME --runtime "$RUNTIME" start
sleep 1
find .
cat $RUNTIME/var/log/$NAME.log
check_ok $NAME --runtime "$RUNTIME" logs
check_ok $NAME --runtime "$RUNTIME" health
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
