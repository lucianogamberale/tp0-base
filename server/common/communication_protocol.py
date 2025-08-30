import json
import logging
from typing import Any

from common import utils


BET_MSG_TYPE = "BET"
ACK_MSG_TYPE = "ACK"

START_DELIMITER = "["
END_DELIMITER = "]"


def decode_bet_batch_message(message: str) -> list[utils.Bet]:
    if not (
        message.startswith(BET_MSG_TYPE + START_DELIMITER)
        and message.endswith(END_DELIMITER)
    ):
        logging.error(
            f"action: decode_bet_batch | result: fail | error: invalid format"
        )
        raise ValueError("Unexpected message bet format")

    bet_batch_data: list = json.loads(message[len(BET_MSG_TYPE) :])
    bet_batch = []
    for bet_data in bet_batch_data:
        bet = utils.Bet(
            agency=bet_data["agency"],
            first_name=bet_data["first_name"],
            last_name=bet_data["last_name"],
            document=bet_data["document"],
            birthdate=bet_data["birthdate"],
            number=bet_data["number"],
        )
        bet_batch.append(bet)
    logging.debug(
        f"action: decode_bet_batch | result: success | count: {len(bet_batch)}"
    )
    return bet_batch


def encode_ack_message(message: str) -> str:
    return ACK_MSG_TYPE + START_DELIMITER + message + END_DELIMITER
