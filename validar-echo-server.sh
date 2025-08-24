#!/bin/sh

network_name='tp0_testing_net'
service_name='nc-test_echo_server'
alpine_image='alpine:3.22'

message="[TEST ECHO SERVER] Message"
command="echo '$message' | nc server 12345"

reply=$(docker run --name=$service_name $alpine_image --network=$network_name --rm sh -c "$command")

if [ "$reply" = "$message" ]; then
  echo 'action: test_echo_server | result: success'
else
  echo 'action: test_echo_server | result: fail'
fi