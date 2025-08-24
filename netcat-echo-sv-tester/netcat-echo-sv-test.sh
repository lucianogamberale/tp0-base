#!/bin/sh

message="[TEST ECHO SERVER] Message"

reply=$(echo "$message" | nc server 12345)
if [ "$reply" = "$message" ]; then
    echo 'action: test_echo_server | result: success'
else
    echo 'action: test_echo_server | result: fail'
fi