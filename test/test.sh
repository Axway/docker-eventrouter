#!/bin/bash
#
# DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS HEADER.
#
# Copyright (c) 2021 Axway Software SA and its affiliates. All rights reserved.
#

stop()
{
  echo
  echo " -- Exit code ${1}; stop all containers"
  docker stop test_er_first test_er_middle test_er_middle_backup test_er_last
  exit ${1}
}

waitstart()
{
  # wait startup
  timeout=40
  i=0
  echo "Waiting for event router startup $i/$timeout..."
  while [ $i -lt $timeout ]; do
    t1_status=$(docker inspect --format='{{json .State.Health.Status}}' test_er_first)
    t2_status=$(docker inspect --format='{{json .State.Health.Status}}' test_er_middle)
    t3_status=$(docker inspect --format='{{json .State.Health.Status}}' test_er_middle_backup)
    t4_status=$(docker inspect --format='{{json .State.Health.Status}}' test_er_last)
    if [ "$t1_status" == "starting" ] || [ "$t2_status" == "starting" ] || [ "$t3_status" == "starting" ] || [ "$t4_status" == "starting" ]; then
      i=$(($i+1))
      sleep 1
      echo "Waiting for event router startup $i/$timeout..."
    else
      i=$timeout
    fi
  done
}

waitstart

# Test First Event Router port
nc -z $FIRST_TARGET_NAME $FIRST_TARGET_PORT
if [ "$?" -ne "0" ]; then
  echo "ERROR: failed to connect to $FIRST_TARGET_NAME:$FIRST_TARGET_PORT"
  stop 1
fi
echo "Successful connection to $FIRST_TARGET_NAME:$FIRST_TARGET_PORT"

# Test Last Event Router port
nc -z $LAST_TARGET_NAME $LAST_TARGET_PORT
if [ "$?" -ne "0" ]; then
  echo "ERROR: failed to connect to $LAST_TARGET_NAME:$LAST_TARGET_PORT"
  stop 1
fi
echo "Successful connection to $LAST_TARGET_NAME:$LAST_TARGET_PORT"

# Test First Event Router port
nc -z $MIDDLE_TARGET_NAME $MIDDLE_TARGET_PORT
if [ "$?" -ne "0" ]; then
  echo "ERROR: failed to connect to $MIDDLE_TARGET_NAME:$MIDDLE_TARGET_PORT"
  stop 1
fi
echo "Successful connection to $MIDDLE_TARGET_NAME:$MIDDLE_TARGET_PORT"

# Test Last Event Router port
nc -z $M_BACKUP_TARGET_NAME $M_BACKUP_TARGET_PORT
if [ "$?" -ne "0" ]; then
  echo "ERROR: failed to connect to $M_BACKUP_TARGET_NAME:$M_BACKUP_TARGET_PORT"
  stop 1
fi
echo "Successful connection to $M_BACKUP_TARGET_NAME:$M_BACKUP_TARGET_PORT"

echo " --- The 4 ERs are started"

echo "TRKPRODUCTNAME=$UA_NAME" > /opt/axway/ua/conf.conf
echo "TRKIDENT=TRKIDENT" >> /opt/axway/ua/conf.conf
echo "TRKTNAME=/opt/axway/ua/TAMPON.dat" >> /opt/axway/ua/conf.conf
echo "TRKTMODE=R" >> /opt/axway/ua/conf.conf
echo "TRKIPADDR=$FIRST_TARGET_NAME" >> /opt/axway/ua/conf.conf
echo "TRKIPPORT=$FIRST_TARGET_PORT" >> /opt/axway/ua/conf.conf
echo "TRKTRACE=0" >> /opt/axway/ua/conf.conf
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

# Wait for message to be sent to 1st ER
timeout=120
i=0
echo "Waiting for 1st ER to be ready $i/$timeout..."
while [ $i -lt $timeout ]; do
  if test -f /opt/axway/data_first/SENTINEL; then
    i=$timeout
  else
    i=$(($i+1))
    sleep 1
    echo "Waiting for 1st ER to be ready $i/$timeout..."
  fi
done

# Sending 1st message
TRKUTIL SendEvent OBJNAME=XFBTransfer,STATE=BEGIN,$EVT
echo "-- 1st message sent"


# Wait for message to be sent to backup
timeout=180
i=0
echo "Waiting for 1st message to be sent to backup $i/$timeout..."
while [ $i -lt $timeout ]; do
  if test "$(wc -c < /opt/axway/data_first/SENTINEL)" -ne "51"; then
    i=$(($i+1))
    sleep 1
    echo "Waiting for 1st message to be sent to backup $i/$timeout..."
  else
    i=$timeout
  fi
done

ls -l /opt/axway/data*

if test "$(wc -c < /opt/axway/data_first/SENTINEL)" -ne "51"; then
    echo "ERROR: Wrong number of characters in buffer file for first eventrouter"
    cat /opt/axway/data_first/SENTINEL
    stop 1
fi

if test "$(wc -c < /opt/axway/data_middle/SENTINEL)" -ne "51"; then
    echo "ERROR: Wrong number of characters in buffer file for middle eventrouter"
    cat /opt/axway/data_middle/SENTINEL
    stop 1
fi

if test "$(wc -c < /opt/axway/data_backup/SENTINEL)" -ne "51"; then
    echo "ERROR: Wrong number of characters in buffer file for backup eventrouter"
    cat /opt/axway/data_backup/SENTINEL
    stop 1
fi

if test "$(wc -c < /opt/axway/data_last/SENTINEL)" -ne "946"; then
    echo "ERROR: Wrong number of characters in buffer file for last eventrouter"
    cat /opt/axway/data_last/SENTINEL
    stop 1
fi

echo "-- Stopping middle eventrouter"
docker stop test_er_middle
sleep 10

docker ps -a | grep test_er_middle

# Sending 2nd message
TRKUTIL SendEvent OBJNAME=XFBTransfer,STATE=BEGIN,$EVT
echo "-- 2nd message sent"

# Wait for message to be sent to 1st ER
timeout=20
i=0
echo "Waiting for 2nd message to be sent to 1st ER $i/$timeout..."
while [ $i -lt $timeout ]; do
  if test "$(wc -c < /opt/axway/data_first/SENTINEL)" -eq "51"; then
    i=$(($i+1))
    sleep 1
    echo "Waiting for 2nd message to be sent to 1st ER $i/$timeout..."
  else
    i=$timeout
  fi
done

# Wait for message to be sent to backup
timeout=180
i=0
echo "Waiting for 2nd message to be sent to backup $i/$timeout..."
while [ $i -lt $timeout ]; do
  if test "$(wc -c < /opt/axway/data_first/SENTINEL)" -ne "51"; then
    i=$(($i+1))
    sleep 1
    echo "Waiting for 2nd message to be sent to backup $i/$timeout..."
  else
    i=$timeout
  fi
done

ls -l /opt/axway/data*

if test "$(wc -c < /opt/axway/data_first/SENTINEL)" -ne "51"; then
    echo "ERROR: Wrong number of characters in buffer file for first eventrouter"
    cat /opt/axway/data_first/SENTINEL
    stop 1
fi

if test "$(wc -c < /opt/axway/data_middle/SENTINEL)" -ne "51"; then
    echo "ERROR: Wrong number of characters in buffer file for middle eventrouter"
    cat /opt/axway/data_middle/SENTINEL
    stop 1
fi

if test "$(wc -c < /opt/axway/data_backup/SENTINEL)" -ne "51"; then
    echo "ERROR: Wrong number of characters in buffer file for backup eventrouter"
    cat /opt/axway/data_backup/SENTINEL
    stop 1
fi

if test "$(wc -c < /opt/axway/data_last/SENTINEL)" -ne "1841"; then
    echo "ERROR: Wrong number of characters in buffer file for last eventrouter"
    cat /opt/axway/data_last/SENTINEL
    stop 1
fi


echo "-- Starting middle eventrouter"
docker start test_er_middle
echo "-- Stopping backup eventrouter"
docker stop test_er_middle_backup

waitstart

docker ps -a | grep test_er_middle

# Sending 3rd message
TRKUTIL SendEvent OBJNAME=XFBTransfer,STATE=BEGIN,$EVT
echo "-- 3rd message sent"

# Wait for message to be sent to 1st ER
timeout=20
i=0
echo "Waiting for 3rd message to be sent to 1st ER $i/$timeout..."
while [ $i -lt $timeout ]; do
  if test "$(wc -c < /opt/axway/data_first/SENTINEL)" -eq "51"; then
    i=$(($i+1))
    sleep 1
    echo "Waiting for 3rd message to be sent to 1st ER $i/$timeout..."
  else
    i=$timeout
  fi
done

# Wait for message to be sent to middle ER
timeout=180
i=0
echo "Waiting for 3rd message to be sent to middle ER $i/$timeout..."
while [ $i -lt $timeout ]; do
  if test "$(wc -c < /opt/axway/data_first/SENTINEL)" -ne "51"; then
    i=$(($i+1))
    sleep 1
    echo "Waiting for 3rd message to be sent to middle ER $i/$timeout..."
  else
    i=$timeout
  fi
done

ls -l /opt/axway/data*

if test "$(wc -c < /opt/axway/data_first/SENTINEL)" -ne "51"; then
    echo "ERROR: Wrong number of characters in buffer file for first eventrouter"
    cat /opt/axway/data_first/SENTINEL
    stop 1
fi

if test "$(wc -c < /opt/axway/data_middle/SENTINEL)" -ne "51"; then
    echo "ERROR: Wrong number of characters in buffer file for middle eventrouter"
    cat /opt/axway/data_middle/SENTINEL
    stop 1
fi

if test "$(wc -c < /opt/axway/data_backup/SENTINEL)" -ne "51"; then
    echo "ERROR: Wrong number of characters in buffer file for backup eventrouter"
    cat /opt/axway/data_backup/SENTINEL
    stop 1
fi

if test "$(wc -c < /opt/axway/data_last/SENTINEL)" -ne "2736"; then
    echo "ERROR: Wrong number of characters in buffer file for last eventrouter"
    cat /opt/axway/data_last/SENTINEL
    stop 1
fi

stop 0
