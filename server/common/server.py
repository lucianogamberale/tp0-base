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

        self.__set_server_as_not_running()

        signal.signal(signal.SIGTERM, self.__sigterm_signal_handler)

    # ============================== PRIVATE - ACCESSING ============================== #

    def __is_running(self) -> bool:
        return self._server_running

    def __set_server_as_not_running(self) -> None:
        self._server_running = False

    def __set_server_as_running(self) -> None:
        self._server_running = True

    # ============================== PRIVATE - SIGNAL HANDLER ============================== #

    def __sigterm_signal_handler(self, signum, frame):
        logging.info("action: sigterm_signal_handler | result: in_progress")

        self.__set_server_as_not_running()

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

    # ============================== PRIVATE - HANDLE CONNECTION ============================== #

    def __handle_client_connection(self, client_connection: socket.socket) -> None:
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be close
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
            logging.info(
                f"action: send_message | result: success | ip: {addr[0]} | msg: {message}"
            )
        except OSError as e:
            logging.error(
                f"action: receive_message | result: fail | error: {e}",
            )
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

        self.__set_server_as_running()
        try:
            while self.__is_running():
                client_connection = self.__accept_new_connection()
                if client_connection is None:
                    continue

                try:
                    self.__handle_client_connection(client_connection)
                finally:
                    client_connection.close()
                    logging.debug("action: client_connection_close | result: success")
        except Exception as e:
            logging.error(f"action: server_run | result: fail | error: {e}")
            raise e
        finally:
            self._server_socket.close()
            logging.debug("action: server_socker_close | result: success")

        logging.info("action: server_shutdown | result: success")
