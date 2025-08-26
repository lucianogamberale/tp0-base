import json
import logging

from common import utils

DELIMITER = "]"

BET_MSG_HEADER = "BET["
ACK_MSG_HEADER = "ACK["


def decode_bet_message(message: str) -> utils.Bet:
    if not (message.startswith(BET_MSG_HEADER) and message.endswith(DELIMITER)):
        logging.error(f"action: receive_bet | result: fail | error: invalid format")
        raise ValueError("Unexpected message bet format")

    bet_data = message[4:-1]
    bet_data = json.loads(bet_data)
    bet = utils.Bet(
        agency=bet_data["agency"],
        first_name=bet_data["first_name"],
        last_name=bet_data["last_name"],
        document=bet_data["document"],
        birthdate=bet_data["birthdate"],
        number=bet_data["number"],
    )
    return bet


def encode_ack_message(message: str) -> str:
    return ACK_MSG_HEADER + message + DELIMITER
