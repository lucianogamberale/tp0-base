#!/bin/bash

# ============================== PRIVATE - UTILS ============================== #

function add-line() {
  local TEXT_TO_APPEND=$1
  echo "$TEXT_TO_APPEND" >> $COMPOSE_FILENAME
}

function add-empty-line() {
  add-line ""
}

# ============================== PRIVATE - NAME BUILDER ============================== #

function add-name() {
  echo "name: tp0" > $COMPOSE_FILENAME
}

# ============================== PRIVATE - SERVICES BUILDER ============================== #

function add-server() {
  add-line "  server:"
  add-line "    container_name: server"
  add-line "    image: server:latest"
  add-line "    entrypoint: python3 /main.py"
  add-line "    environment:"
  add-line "      - PYTHONUNBUFFERED=1"
  add-line "      - LOGGING_LEVEL=DEBUG"
  add-line "    networks:"
  add-line "      - testing_net"
}

function add-client() {
  local CLIENT_ID=$1

  add-line "  client$CLIENT_ID:"
  add-line "    container_name: client$CLIENT_ID"
  add-line "    image: client:latest"
  add-line "    entrypoint: /client"
  add-line "    environment:"
  add-line "      - CLI_ID=$CLIENT_ID"
  add-line "      - CLI_LOG_LEVEL=DEBUG"
  add-line "    networks:"
  add-line "      - testing_net"
  add-line "    depends_on:"
  add-line "      - server"
}

function add-services() {
  add-line "services:"
  add-server
  for (( i=1; i<=CLIENTS_AMOUNT; i++ )); do
    add-empty-line
    add-client $i
  done
}

# ============================== PRIVATE - NETWORKS BUILDER ============================== #

function add-networks() {
  add-line "networks:"
  add-line "  testing_net:"
  add-line "    ipam:"
  add-line "      driver: default"
  add-line "      config:"
  add-line "        - subnet: 172.25.125.0/24"
}

# ============================== PRIVATE - DOCKER COMPOSE FILE BUILDER ============================== #

function build-docker-compose-file() {
  echo "Generando archivo $COMPOSE_FILENAME con $CLIENTS_AMOUNT cliente(s) ..."
  
  add-name
  add-services
  add-empty-line
  add-networks

  echo "Generando archivo $COMPOSE_FILENAME con $CLIENTS_AMOUNT cliente(s) [DONE]"
}

# ============================== MAIN ============================== #

if [ $# -ne 2 ]; then
  echo "Uso: $0 <compose_filename.yaml> <clients_amount>"
  echo "Ejemplo: $0 docker-compose-dev.yaml 5"
  exit 1
fi

COMPOSE_FILENAME=$1
CLIENTS_AMOUNT=$2

if ! [[ "$CLIENTS_AMOUNT" =~ ^[0-9]+$ ]] || [ "$CLIENTS_AMOUNT" -lt 1 ]; then
  echo "Error: La cantidad de clientes debe ser un entero positivo."
  exit 1
fi

echo "Nombre del archivo de salida: $COMPOSE_FILENAME"
echo "Cantidad de clientes: $CLIENTS_AMOUNT"

build-docker-compose-file
