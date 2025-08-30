#!/bin/bash

# ============================== PRIVATE - UTILS ============================== #

function add-line() {
  local compose_filename=$1
  local text=$2

  echo "$text" >> "$compose_filename"
}

function add-empty-line() {
  local compose_filename=$1

  add-line $compose_filename ""
}

# ============================== PRIVATE - NAME BUILDER ============================== #

function add-name() {
  local compose_filename=$1
  
  echo "name: tp0" > "$compose_filename"
}

# ============================== PRIVATE - SERVICES BUILDER ============================== #

function add-server-service() {
  local compose_filename=$1

  add-line $compose_filename "  server:"
  add-line $compose_filename "    container_name: server"
  add-line $compose_filename "    image: server:latest"
  add-line $compose_filename "    entrypoint: python3 /main.py"
  add-line $compose_filename "    environment:"
  add-line $compose_filename "      - PYTHONUNBUFFERED=1"
  add-line $compose_filename "    networks:"
  add-line $compose_filename "      - testing_net"
  add-line $compose_filename "    volumes:"
  add-line $compose_filename "      - type: bind"
  add-line $compose_filename "        source: ./server/config.ini"
  add-line $compose_filename "        target: /config.ini"
  add-line $compose_filename "        read_only: true"
}

function add-client-service() {
  local compose_filename=$1
  local client_id=$2

  add-line $compose_filename "  client$client_id:"
  add-line $compose_filename "    container_name: client$client_id"
  add-line $compose_filename "    image: client:latest"
  add-line $compose_filename "    entrypoint: /client"
  add-line $compose_filename "    environment:"
  add-line $compose_filename "      - CLI_ID=$client_id"
  add-line $compose_filename "    networks:"
  add-line $compose_filename "      - testing_net"
  add-line $compose_filename "    volumes:"
  add-line $compose_filename "      - type: bind"
  add-line $compose_filename "        source: ./client/config.yaml"
  add-line $compose_filename "        target: /config.yaml"
  add-line $compose_filename "        read_only: true"
  add-line $compose_filename "      - type: bind"
  add-line $compose_filename "        source: ./.data/agency-$client_id.csv"
  add-line $compose_filename "        target: /agency-$client_id.csv"
  add-line $compose_filename "        read_only: true"
  add-line $compose_filename "    deploy:"
  add-line $compose_filename "      restart_policy:"
  add-line $compose_filename "        condition: on-failure"
  add-line $compose_filename "        delay: 5s"
  add-line $compose_filename "        max_attempts: 1"
  add-line $compose_filename "    depends_on:"
  add-line $compose_filename "      - server"
}

function add-services() {
  local compose_filename=$1

  add-line $compose_filename "services:"

  add-server-service $compose_filename
  
  for (( i=1; i<=clients_amount; i++ )); do
    add-empty-line $compose_filename
    add-client-service $compose_filename $i
  done
}

# ============================== PRIVATE - NETWORKS BUILDER ============================== #

function add-networks() {
  local compose_filename=$1

  add-line $compose_filename "networks:"
  add-line $compose_filename "  testing_net:"
  add-line $compose_filename "    ipam:"
  add-line $compose_filename "      driver: default"
  add-line $compose_filename "      config:"
  add-line $compose_filename "        - subnet: 172.25.125.0/24"
}

# ============================== PRIVATE - DOCKER COMPOSE FILE BUILDER ============================== #

function build-docker-compose-file() {
  local compose_filename=$1
  local clients_amount=$2

  echo "Generando archivo $compose_filename con $clients_amount cliente(s) ..."
  
  add-name $compose_filename
  add-services $compose_filename $clients_amount
  add-empty-line $compose_filename
  add-networks $compose_filename

  echo "Generando archivo $compose_filename con $clients_amount cliente(s) ... [DONE]"
}

# ============================== MAIN ============================== #

if [ $# -ne 2 ]; then
  echo "Uso: $0 <compose_filename.yaml> <clients_amount>"
  echo "Ejemplo: $0 docker-compose-dev.yaml 5"
  exit 1
fi

compose_filename_param=$1
clients_amount_param=$2

if ! [[ "$clients_amount_param" =~ ^[0-9]+$ ]] || [ "$clients_amount_param" -lt 0 ]; then
  echo "Error: La cantidad de clientes debe ser un entero mayor o igual a cero."
  exit 1
fi

echo "Nombre del archivo de salida: $compose_filename_param"
echo "Cantidad de clientes: $clients_amount_param"

build-docker-compose-file $compose_filename_param $clients_amount_param