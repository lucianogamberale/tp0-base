#!/bin/sh

network_name='tp0_testing_net'
service_name='nc-test_echo_server'
alpine_image='alpine:3.22'

message="[TEST ECHO SERVER] Message"

docker run -dit --name=$service_name --network=$network_name $alpine_image sh

reply=$(docker exec $service_name sh -c "echo '$message' | nc server 12345")
if [ "$reply" = "$message" ]; then
    docker exec $service_name sh -c "echo 'action: test_echo_server | result: success'"
else
    docker exec $service_name sh -c "echo 'action: test_echo_server | result: fail'"
fi

docker stop $service_name
docker rm $service_name