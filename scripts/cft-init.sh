#!/bin/bash
#
VERSION=0.0.1
INSTALL=$(dirname "$0")

#SENTINEL_HOST=ingestion-lumberjack.branch2.trcblt.com
#SENTINEL_PORT=80

SENTINEL_HOST=qlt
SENTINEL_PORT=3333
if [ -f "$INSTALL/cli-common.sh" ]; then
. "$INSTALL/cli-common.sh"
elif [ -f "$HOME/bin/cli-common.sh" ]; then
. $HOME/bin/cli-common.sh
fi

#trap 'echo "cleanup $TMPDIR..."; ls "$TMPDIR" ; rm -rf "$TMPDIR"' EXIT
trap 'rm -rf "$TMPDIR"' EXIT 
TMPDIR=$(mktemp -d) || exit 1
tmpfile() {
    mktemp "$TMPDIR/tmp".XXXXX || exit 1
}

info() {
    echo "$@"
    "$@"
}
cft() {
    local svc="$1"
    shift
    echo "$svc $@"
    info docker-compose exec "$svc" /bin/bash -c "cd ./data/runtime; . ./profile; ls; $@"
    echo $?
}

cft2script() {
    local id
    local fname
    TMPFILE=$(tmpfile)
    chmod a+r $TMPFILE

    echo "cd ./data/runtime; . ./profile" >"$TMPFILE"
    cat >>"$TMPFILE"
    id=$(docker-compose ps -q $1)
    info docker cp "$TMPFILE" $id:/home/cft/zou.sh
    info docker exec $id cat /home/cft/zou.sh
    info docker exec $id bash /home/cft/zou.sh
}

case ${1:-} in
    down)  ## down
        docker-compose down -v
    ;;
    up) ## up
        docker-compose up --build -d
    ;;
    
    up-all) ## up-all
        $0 down
        $0 up
        $0 init-sentinel
        $0 init-part
        $0 run
        $0 listcat
    ;;

    init-sentinel) ## init-sentinel
        TMPFILE=$(tmpfile)
        chmod a+r $TMPFILE

        cat >$TMPFILE <<EOF
CFTUTIL UCONFSET ID=sentinel.xfb.enable, value=Yes
CFTUTIL UCONFSET ID=sentinel.trkipaddr, value=$SENTINEL_HOST
CFTUTIL UCONFSET ID=sentinel.trkipport, value=$SENTINEL_PORT
cft restart
EOF
        cft2script cft1 <$TMPFILE
        cft2script cft2 <$TMPFILE
        cft2script cft3 <$TMPFILE
    ;;

    init-part) ## init parts

#cft1
cft2script cft1 <<EOF
CFTUTIL CFTPART ID=APP3, nspart=app1, nrpart=app3, SAP=1761, PROT=TCP1_PESIT1, IPART=cft2, OMAXTIME=0, OMINTIME=0,  MODE=REPLACE

CFTUTIL CFTPART ID=cft2, nspart=cft1, nrpart=cft2, SAP=1761, PROT=TCP1_PESIT1, MODE=REPLACE
CFTUTIL CFTTCP ID=cft2, HOST=cft2, MODE=REPLACE

CFTUTIL CFTPART ID=cft3, nspart=cft1, nrpart=cft3, SAP=1761, PROT=TCP1_PESIT1, MODE=REPLACE
CFTUTIL CFTTCP ID=cft3, HOST=cft3, MODE=REPLACE
EOF

#cft2
cft2script cft2 <<EOF

CFTUTIL CFTPART ID=APP3, nspart=app1, nrpart=app3, SAP=1761, PROT=TCP1_PESIT1, OMAXTIME=0, OMINTIME=0, IPART=cft3, MODE=REPLACE,
CFTUTIL CFTTCP ID=APP3, HOST=cft3, MODE=REPLACE

CFTUTIL CFTPART ID=cft1, nspart=cft2, nrpart=cft1, SAP=1761, PROT=TCP1_PESIT1, MODE=REPLACE
CFTUTIL CFTTCP ID=cft1, HOST=cft1, MODE=REPLACE

CFTUTIL CFTPART ID=cft3, nspart=cft2, nrpart=cft3, SAP=1761, PROT=TCP1_PESIT1, MODE=REPLACE
CFTUTIL CFTTCP ID=cft3, HOST=cft3, MODE=REPLACE
EOF

#cft3
cft2script cft3 <<EOF
CFTUTIL CFTPART ID=APP1, nspart=app3, nrpart=cft1, PROT=TCP1_PESIT1, MODE=REPLACE

CFTUTIL CFTPART ID=CFT1, nspart=cft3, nrpart=cft1, SAP=1761, PROT=TCP1_PESIT1, OMAXTIME=0, OMINTIME=0, IPART=cft2, MODE=REPLACE
CFTUTIL CFTTCP ID=CFT1, HOST=cft1, MODE=REPLACE

CFTUTIL CFTPART ID=CFT2, nspart=cft3, nrpart=cft2, SAP=1761, PROT=TCP1_PESIT1, MODE=REPLACE
CFTUTIL CFTTCP ID=CFT2, HOST=cft2, MODE=REPLACE
EOF
    ;;
    run) ## run

cft2script cft1 <<EOF
    CFTUTIL send part=paris
    CFTUTIL send part=app3
    CFTUTIL send part=cft2
    CFTUTIL send part=cft3
EOF

cft2script cft2 <<EOF
    CFTUTIL send part=paris
    CFTUTIL send part=cft1
    CFTUTIL send part=cft3
EOF

cft2script cft3 <<EOF
    CFTUTIL send part=paris
    CFTUTIL send part=cft1
    CFTUTIL send part=cft2
    CFTUTIL send part=part-error
EOF

    ;;
    run-accross) ## run-accross
cft2script cft1 <<EOF
    CFTUTIL send part=app3
EOF
    ;;
    listcat)
        cft2script cft1 <<EOF
CFTUTIL LISTCAT
EOF
        cft2script cft2 <<EOF
CFTUTIL LISTCAT
EOF
        cft2script cft3 <<EOF
CFTUTIL LISTCAT
EOF

    ;;
    cftutil) ## cftutil ## <cft> <cmd...>
    shift
    cft=$1
    shift
    cft2script $cft <<EOF
CFTUTIL $@
EOF
    ;;
    *)
        _cli_helper "$@"
    ;;
esac

