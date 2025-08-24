#!/bin/sh

server_service_name='server'
server_port=12345
message="[TEST ECHO SERVER] Message"

reply=$(echo "$message" | nc $server_service_name $server_port)
if [ "$reply" = "$message" ]; then
    echo 'action: test_echo_server | result: success'
else
    echo 'action: test_echo_server | result: fail'
fi