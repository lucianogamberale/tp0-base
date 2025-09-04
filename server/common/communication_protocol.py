import logging

from common import utils

MESSAGE_TYPE_LENGTH = 3

ACK_MSG_TYPE = "ACK"
BET_MSG_TYPE = "BET"
NO_MORE_BETS_MSG_TYPE = "NMB"
ASK_FOR_WINNERS_MSG_TYPE = "ASK"
WAIT_MSG_TYPE = "WIT"
WINNERS_MSG_TYPE = "WIN"

START_MSG_DELIMITER = "["
END_MSG_DELIMITER = "]"

START_BET_DELIMITER = "{"
END_BET_DELIMITER = "}"
BET_BATCH_SEPARATOR = ";"
BET_FIELDS_SEPARATOR = ","

WINNERS_SEPARATOR = ","

# ============================= DECODE ============================== #


def decode_message_type(message: str) -> str:
    if len(message) < MESSAGE_TYPE_LENGTH:
        raise ValueError("Message too short to contain a valid message type")
    return message[:MESSAGE_TYPE_LENGTH]


def __assert_message_format(message: str, expected_message_type: str) -> None:
    received_message_type = decode_message_type(message)
    if received_message_type != expected_message_type:
        raise ValueError(
            f"Unexpected message type. Expected: {expected_message_type}, Received: {received_message_type}",
        )

    if not (
        message.startswith(expected_message_type + START_MSG_DELIMITER)
        and message.endswith(END_MSG_DELIMITER)
    ):
        raise ValueError("Unexpected message format")


def __get_message_payload(message: str) -> str:
    # remove message type
    payload = message[MESSAGE_TYPE_LENGTH:]

    # remove message delimiters
    payload = payload[len(START_MSG_DELIMITER) : -len(END_MSG_DELIMITER)]

    return payload


def __decode_field(key_value_pair: str) -> tuple[str, str]:
    key, value = key_value_pair.split(":", 1)
    key = key.strip('"')
    value = value.strip('"')
    return key, value


def __decode_bet(payload: str) -> utils.Bet:
    # INPUT: {"agency":"1","first_name":"John","last_name":"Doe","document":"12345678","birthdate":"1990-01-01","number":"42"}
    payload = payload.strip(START_BET_DELIMITER)
    payload = payload.strip(END_BET_DELIMITER)

    key_value_pairs = payload.split(BET_FIELDS_SEPARATOR)

    bet_data = {}
    for key_value_pair in key_value_pairs:
        key, value = __decode_field(key_value_pair)
        bet_data[key] = value

    bet = utils.Bet(
        agency=bet_data["agency"],
        first_name=bet_data["first_name"],
        last_name=bet_data["last_name"],
        document=bet_data["document"],
        birthdate=bet_data["birthdate"],
        number=bet_data["number"],
    )
    return bet


def decode_bet_batch_message(message: str) -> list[utils.Bet]:
    # INPUT: BET[{"agency": "001","first_name":"John","last_name":"Doe","document":"12345678","birthdate":"1990-01-01","number": 42};{...}; ...]
    __assert_message_format(message, BET_MSG_TYPE)
    payload = __get_message_payload(message)
    bet_entries = payload.split(BET_BATCH_SEPARATOR)

    bet_batch = []
    for bet_entry in bet_entries:
        bet = __decode_bet(bet_entry)
        bet_batch.append(bet)

    return bet_batch


def decode_no_more_bets_message(message: str) -> int:
    # INPUT: NMB["agency":"1"]
    __assert_message_format(message, NO_MORE_BETS_MSG_TYPE)
    payload = __get_message_payload(message)

    _, agency = __decode_field(payload)

    return int(agency)


def decode_ask_for_winners_message(message: str) -> int:
    # INPUT: ASK["agency":"1"]
    __assert_message_format(message, ASK_FOR_WINNERS_MSG_TYPE)
    payload = __get_message_payload(message)

    _, agency = __decode_field(payload)

    return int(agency)


# ============================= ENCODE ============================== #


def __encode_message(message_type: str, payload: str) -> str:
    encoded_payload = message_type
    encoded_payload += START_MSG_DELIMITER
    encoded_payload += payload
    encoded_payload += END_MSG_DELIMITER
    return encoded_payload


def encode_ack_message(message: str) -> str:
    return __encode_message(ACK_MSG_TYPE, message)


def encode_winners_message(winners: list[utils.Bet]) -> str:
    # OUTPUT: WIN["12135000","87654321", ...]
    encoded_payload = [f'"{winner.document}"' for winner in winners]
    encoded_payload = WINNERS_SEPARATOR.join(encoded_payload)
    return __encode_message(WINNERS_MSG_TYPE, encoded_payload)


def encode_wait_message() -> str:
    return __encode_message(WAIT_MSG_TYPE, "")
