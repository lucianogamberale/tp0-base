#!/bin/sh

image_tag='netcat-echo-sv-tester:latest'
service_name='netcat-echo-sv-tester'
network_name='tp0_testing_net'
script_name='netcat-echo-sv-test.sh'

docker build --tag=$image_tag ./netcat-echo-sv-tester

docker run \
    --rm \
    --name=$service_name \
    --network=$network_name \
    "$image_tag" sh -c "./$script_name"