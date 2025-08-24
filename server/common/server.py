import os
import signal
import socket
import logging
from typing import Optional


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
        self._server_socket.close()

        logging.info("action: sigterm_signal_handler | result: success")

    # ============================== PRIVATE - ACCEPT CONNECTION ============================== #

    def __accept_new_connection(self) -> Optional[socket.socket]:
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """
        client_connection = None

        try:
            logging.info("action: accept_connections | result: in_progress")
            client_connection, addr = self._server_socket.accept()
            # os.kill(os.getpid(), signal.SIGTERM)  # For testing purposes
            logging.info(
                f"action: accept_connections | result: success | ip: {addr[0]}"
            )
        except OSError as e:
            if client_connection is not None:
                client_connection.shutdown(socket.SHUT_RDWR)
                client_connection.close()
            client_connection = None
            logging.error(f"action: accept_connections | result: fail | error: {e}")

        return client_connection

    # ============================== PRIVATE - HANDLE CONNECTION ============================== #

    def __handle_client_connection(self, client_connection: socket.socket) -> None:
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            # TODO: Modify the receive to avoid short-reads
            message = client_connection.recv(1024).rstrip().decode("utf-8")

            addr = client_connection.getpeername()
            logging.info(
                f"action: receive_message | result: success | ip: {addr[0]} | msg: {message}"
            )

            # TODO: Modify the send to avoid short-writes
            client_connection.send("{}\n".format(message).encode("utf-8"))
        except OSError as e:
            logging.error(f"action: receive_message | result: fail | error: {e}")
        finally:
            client_connection.shutdown(socket.SHUT_RDWR)
            client_connection.close()

    # ============================== PUBLIC ============================== #

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communucation
        finishes, servers starts to accept new connections again
        """
        self._server_running = True

        while self._server_running:
            client_connection = self.__accept_new_connection()
            if client_connection is not None:
                self.__handle_client_connection(client_connection)
