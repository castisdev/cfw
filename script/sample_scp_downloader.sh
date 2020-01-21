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
BPS=$4

#
# https://stackoverflow.com/questions/20627207/check-the-status-code-of-a-scp-command-and-if-it-is-failed-then-call-scp-on-ano
# https://www.lesstif.com/pages/viewpage.action?pageId=12943452
# - id/password 등의 입력을 피하기 위해서는
# 	local server에 private key,public key를 만들고
#		source server에 public key를 미리 copy 해놓고 사용해야한다고 함
#		예:
#			$> ssh-keygen -t rsa
#			$> ssh-copy-id yourid@sourceserverip
#
# - scp 는 -l option에 Kbps를 사용할 수 있음
# - scp 의 결과값 (0 이면 성공)을 받아서 성공 메시지를
# 	stderr로 출력해주어야 cfw와 연동이 됨
#		성공 메시지에는 cfw의 [downloader_download_success_match_string] 값을
#		포함해야 함
KBPS=$(($BPS/1000))

scp -i ~/.ssh/id_rsa -l $KBPS $SRC_IP:$SRC_FILEPATH $FILENAME
if [ $? -eq 0 ]
then
	echo successful >&2
else
	echo fail >&2
fi
