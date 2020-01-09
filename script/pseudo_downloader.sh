#!/bin/bash
if [ -n "$2" ]; then
	:
else
	echo -e "no filename"
	exit
fi

SRC_IP=$1
FILENAME=$2
SRC_FILEPATH=$3
COPYSPEED=$4

echo "download" $SRC_IP, $FILENAME, $SRC_FILEPATH, $COPYSPEED
touch $FILENAME
sleep 5

>&2 echo "Successfully download" $FILENAME
