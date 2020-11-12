#!/bin/bash
#
# DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS HEADER.
#
# Copyright (c) 2019 Axway Software SA and its affiliates. All rights reserved.
#
set -euo pipefail

sleep 30

# Test Event Router 1 port
nc -z $TARGET_NAME $TARGET_PORT
if [ "$?" -ne "0" ]; then
  echo "ERROR: failed to connect to $TARGET_NAME:$TARGET_PORT"
  exit 1
fi
echo "Successful connection to $TARGET_NAME:$TARGET_PORT"

# Test Event Router 2 port
nc -z $TARGET2_NAME $TARGET2_PORT
if [ "$?" -ne "0" ]; then
  echo "ERROR: failed to connect to $TARGET2_NAME:$TARGET2_PORT"
  exit 1
fi
echo "Successful connection to $TARGET2_NAME:$TARGET2_PORT"

echo "TRKPRODUCTNAME=$UA_NAME" > /opt/axway/ua/conf.conf
echo "TRKIDENT=TRKIDENT" >> /opt/axway/ua/conf.conf
echo "TRKTNAME=/opt/axway/ua/TAMPON.dat" >> /opt/axway/ua/conf.conf
echo "TRKTMODE=R" >> /opt/axway/ua/conf.conf
echo "TRKIPADDR=$TARGET_NAME" >> /opt/axway/ua/conf.conf
echo "TRKIPPORT=$TARGET_PORT" >> /opt/axway/ua/conf.conf
echo "TRKTRACE=1" >> /opt/axway/ua/conf.conf
echo "TRKSENTCODE=2" >> /opt/axway/ua/conf.conf
echo "TRKMSGENCODING=UTF-8" >> /opt/axway/ua/conf.conf

cd ua

sed -i 's/ua_dir=$PWD/ua_dir=\/opt\/axway\/ua/g' profile
export LD_LIBRARY_PATH=""

. profile

TRKUTIL about

EVT="APPLICATION=LOOPAPPL,CYCLEID=LOOPT"
EVT="$EVT,SENDERID=SND,RECEIVERID=RCV"
EVT="$EVT,FILENAME=ABCDEF"
EVT="$EVT,COMPRESSION=2"
EVT="$EVT,config=/opt/axway/ua/conf.conf"

TRKUTIL SendEvent OBJNAME=XFBTransfer,STATE=BEGIN,$EVT

sleep 30

ls -l /opt/axway/data*

if test "$(wc -c < /opt/axway/data1/DEFAULT)" -ne "51"; then
    echo "ERROR: Wrong number of characters in buffer file for eventrouter 1"
    cat /opt/axway/data1/DEFAULT
    exit 1
fi


if test "$(wc -c < /opt/axway/data2/DEFAULT)" -ne "946"; then
    echo "ERROR: Wrong number of characters in buffer file for eventrouter 2"
    cat /opt/axway/data2/DEFAULT
    exit 1
fi


