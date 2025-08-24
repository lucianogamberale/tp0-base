#!/bin/sh

network_name='tp0_testing_net'

image_tag='netcat-echo-sv-tester:latest'
script_name='netcat-echo-sv-test.sh'

docker build --tag=$image_tag ./netcat-echo-sv-tester
docker run --rm --network=$network_name $image_tag sh -c "./$script_name"