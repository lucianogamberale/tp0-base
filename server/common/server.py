import signal
import socket
import logging
import json
from typing import Optional

from common import utils, communication_protocol


class Server:

    # ============================== INITIALIZE ============================== #

    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(("", port))
        self._server_socket.listen(listen_backlog)

        self._server_running = False
        signal.signal(signal.SIGTERM, self.__sigterm_signal_handler)

    # ============================== PRIVATE - SIGNAL HANDLER ============================== #

    def __sigterm_signal_handler(self, signum, frame):
        logging.info("action: sigterm_signal_handler | result: in_progress")

        self._server_running = False

        self._server_socket.shutdown(socket.SHUT_RDWR)
        self._server_socket.close()
        logging.debug("action: sigterm_server_socket_close | result: success")

        logging.info("action: sigterm_signal_handler | result: success")

    # ============================== PRIVATE - ACCEPT CONNECTION ============================== #

    def __accept_new_connection(self) -> Optional[socket.socket]:
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        client_connection: Optional[socket.socket] = None
        try:
            logging.info(
                "action: accept_connections | result: in_progress",
            )
            client_connection, addr = self._server_socket.accept()
            logging.info(
                f"action: accept_connections | result: success | ip: {addr[0]}",
            )
            return client_connection
        except OSError as e:
            if client_connection is not None:
                client_connection.shutdown(socket.SHUT_RDWR)
                client_connection.close()
                logging.debug("action: client_connection_close | result: success")
            logging.error(f"action: accept_connections | result: fail | error: {e}")
            return None

    # ============================== PRIVATE - SEND/RECEIVE MESSAGES ============================== #

    def __send_message(self, client_connection: socket.socket, message: str) -> None:
        logging.debug(f"action: send_message | result: in_progress | msg: {message}")

        client_connection.sendall(message.encode("utf-8"))

        logging.debug(f"action: send_message | result: success |  msg: {message}")

    def __receive_message(self, client_connection: socket.socket) -> str:
        logging.debug(f"action: receive_message | result: in_progress")

        buffsize = utils.KiB
        bytes_received = b""

        all_data_received = False
        while not all_data_received:
            chunk = client_connection.recv(buffsize)
            if len(chunk) == 0:
                logging.error(
                    f"action: receive_message | result: fail | error: unexpected disconnection",
                )
                OSError("Unexpected disconnection of the client")

            logging.debug(f"action: receive_chuck | result: success | chunk: {chunk}")
            if chunk.endswith(communication_protocol.DELIMITER.encode("utf-8")):
                all_data_received = True

            bytes_received += chunk

        message = bytes_received.decode("utf-8")
        logging.debug(f"action: receive_message | result: success | msg: {message}")
        return message

    # ============================== PRIVATE - HANDLE BET ============================== #

    def __receive_bet(self, client_connection: socket.socket) -> utils.Bet:
        logging.info(f"action: receive_bet | result: in_progress")

        message = self.__receive_message(client_connection)
        bet = communication_protocol.decode_bet_message(message)

        logging.info(f"action: receive_bet | result: success")
        return bet

    def __send_bet_ack(self, client_connection: socket.socket) -> None:
        logging.info(f"action: send_bet_ack | result: success")

        message = communication_protocol.encode_ack_message("1")
        self.__send_message(client_connection, message)

        logging.info(f"action: send_bet_ack | result: success")

    # ============================== PRIVATE - HANDLE CONNECTION ============================== #

    def __handle_client_connection(self, client_connection: socket.socket) -> None:
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            bet = self.__receive_bet(client_connection)

            utils.store_bets([bet])
            logging.info(
                f"action: apuesta_almacenada | result: success | dni: {bet.document} | numero: {bet.number}"
            )

            self.__send_bet_ack(client_connection)
        except Exception as e:
            logging.error(f"action: handle_connection | result: fail | error: {e}")
            raise e

    # ============================== PUBLIC ============================== #

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """
        logging.info("action: server_startup | result: success")

        self._server_running = True
        with self._server_socket:
            while self._server_running:
                client_connection = self.__accept_new_connection()
                if client_connection is None:
                    continue

                with client_connection:
                    self.__handle_client_connection(client_connection)
                    logging.debug("action: client_connection_close | result: success")
            logging.debug("action: server_socker_close | result: success")

        logging.info("action: server_shutdown | result: success")
