import signal
import socket
import logging
import threading
from typing import Optional

from common import utils, communication_protocol


def handle_thread_exeptions(args):
    logging.error(f"action: thread_exception | result: fail | error: {args.exc_value}")
    raise args.exc_value


class Server:

    # ============================== INITIALIZE ============================== #

    def __init__(self, port, listen_backlog, number_of_agencies: int) -> None:
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(("", port))
        self._server_socket.listen(listen_backlog)

        self._number_of_agencies = number_of_agencies

        self._server_running = threading.Event()
        self.__set_server_as_not_running()
        signal.signal(signal.SIGTERM, self.__sigterm_signal_handler)

        self._agencies_information = {}

        self._spawned_threads: list[threading.Thread] = []
        threading.excepthook = handle_thread_exeptions
        self._draw_barrier = threading.Barrier(number_of_agencies)

    # ============================== PRIVATE - ACCESSING ============================== #

    def __is_running(self) -> bool:
        return self._server_running.is_set()

    def __set_server_as_not_running(self) -> None:
        self._server_running.clear()

    def __set_server_as_running(self) -> None:
        self._server_running.set()

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
                raise OSError("Unexpected disconnection of the client")

            logging.debug(
                f"action: receive_chunk | result: success | chunk size: {len(chunk)}"
            )
            if chunk.endswith(communication_protocol.END_MSG_DELIMITER.encode("utf-8")):
                all_data_received = True

            bytes_received += chunk

        message = bytes_received.decode("utf-8")
        logging.debug(f"action: receive_message | result: success | msg: {message}")
        return message

    # ============================== PRIVATE - AGENCIES INFORMATION ============================== #

    def __all_winners_sent(self) -> bool:
        return (
            self._draw_barrier.broken
            or self._draw_barrier.n_waiting == self._number_of_agencies
        )

    # ============================== PRIVATE - SEND ACK ============================== #

    def __send_ack_message(
        self, client_connection: socket.socket, message: str, logging_action: str
    ) -> None:
        logging.info(f"action: {logging_action} | result: in_progress")

        message = communication_protocol.encode_ack_message(message)
        self.__send_message(client_connection, message)

        logging.info(f"action: {logging_action} | result: success")

    # ============================== PRIVATE - HANDLE BET BATCH ============================== #

    def __send_bet_batch_ack(
        self, client_connection: socket.socket, batch_size: int
    ) -> None:
        self.__send_ack_message(
            client_connection, str(batch_size), "send_bet_batch_ack"
        )

    def __handle_bet_batch_message(
        self, client_connection: socket.socket, message: str
    ) -> None:
        bet_batch = []
        try:
            logging.info(f"action: receive_bet_batch | result: in_progress")
            bet_batch = communication_protocol.decode_bet_batch_message(message)
            logging.info(f"action: receive_bet_batch | result: success")

            if len(bet_batch) == 0:
                raise ValueError("Empty bet batch received")

            utils.store_bets(bet_batch)
            self.__send_bet_batch_ack(client_connection, len(bet_batch))
            logging.info(
                f"action: apuesta_recibida | result: success | cantidad: {len(bet_batch)}"
            )
        except Exception as e:
            self.__send_bet_batch_ack(client_connection, 0)
            logging.info(
                f"action: apuesta_recibida | result: fail | cantidad: {len(bet_batch)}"
            )
            raise e

    # ============================== PRIVATE - HANDLE NO MORE BETS ============================== #

    def __handle_no_more_bets_message(
        self, client_connection: socket.socket, message: str
    ) -> None:
        logging.info(f"action: receive_no_more_bets | result: in_progress")
        communication_protocol.decode_no_more_bets_message(message)
        logging.info(f"action: receive_no_more_bets | result: success")

        self.__send_ack_message(
            client_connection,
            communication_protocol.NO_MORE_BETS_MSG_TYPE,
            "ack_no_more_bets",
        )

    # ============================== PRIVATE - HANDLE ASK FOR WINNERS ============================== #

    def __send_winners(self, client_connection: socket.socket, agency: int) -> None:
        winners = [
            bet
            for bet in utils.load_bets()
            if bet.agency == agency and utils.has_won(bet)
        ]
        message = communication_protocol.encode_winners_message(winners)
        self.__send_message(client_connection, message)
        logging.info(
            f"action: send_winners | result: success | agency: {agency} | winners: {len(winners)}",
        )

    def __handle_ask_for_winners(
        self, client_connection: socket.socket, message: str
    ) -> None:
        logging.info(f"action: receive_ask_for_winners | result: in_progress")
        agency = communication_protocol.decode_ask_for_winners_message(message)
        logging.info(f"action: receive_ask_for_winners | result: success")

        if self._draw_barrier.wait() == 0:
            logging.info("action: sorteo | result: success")

        self.__send_winners(client_connection, agency)

    # ============================== PRIVATE - HANDLE CONNECTION ============================== #

    def __handle_client_connection(self, client_connection: socket.socket) -> None:
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """

        while self.__is_running() and not self.__all_winners_sent():
            message = self.__receive_message(client_connection)

            if message.startswith(communication_protocol.BET_MSG_TYPE):
                self.__handle_bet_batch_message(client_connection, message)
            elif message.startswith(communication_protocol.NO_MORE_BETS_MSG_TYPE):
                self.__handle_no_more_bets_message(client_connection, message)
            elif message.startswith(communication_protocol.ASK_FOR_WINNERS_MSG_TYPE):
                self.__handle_ask_for_winners(client_connection, message)
                break
            else:
                raise ValueError(
                    f'Invalid message type received from client "{communication_protocol.decode_message_type(message)}"'
                )

    def __handle_client_connection_thread_target(
        self, client_connection: socket.socket
    ) -> None:
        try:
            logging.debug(
                "action: handle_client_connection | result: in_progress",
            )
            self.__handle_client_connection(client_connection)
            logging.debug(
                "action: handle_client_connection | result: success",
            )
        except Exception as e:
            logging.error(
                f"action: handle_client_connection | result: fail | error: {e}"
            )
            raise e
        finally:
            client_connection.close()
            logging.debug("action: client_connection_close | result: success")

    # ============================== PRIVATE - HANDLE THREADS ============================== #

    def __handle_client_connection_using_a_thread(
        self, client_connection: socket.socket
    ) -> None:
        thread = threading.Thread(
            target=self.__handle_client_connection_thread_target,
            args=(client_connection,),
        )
        self._spawned_threads.append(thread)
        thread.start()
        logging.debug("action: handle_client_connection_thread_start | result: success")

    def __join_all_threads(self) -> None:
        for thread in self._spawned_threads:
            thread.join()
        logging.debug("action: thread_join | result: success")

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
            while self.__is_running() and not self.__all_winners_sent():
                client_connection = self.__accept_new_connection()
                if client_connection is None:
                    continue

                try:
                    self.__handle_client_connection_using_a_thread(client_connection)
                except Exception as e:
                    client_connection.close()
                    logging.debug("action: client_connection_close | result: success")
                    raise e
        except Exception as e:
            logging.error(f"action: server_run | result: fail | error: {e}")
            raise e
        finally:
            self.__join_all_threads()
            self._server_socket.close()
            logging.debug("action: server_socker_close | result: success")

        logging.info("action: server_shutdown | result: success")
