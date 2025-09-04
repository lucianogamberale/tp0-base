package common

import (
	"fmt"
)

const (
	ACK_MSG_TYPE = "ACK"
	BET_MSG_TYPE = "BET"

	START_MSG_DELIMITER = "["
	END_MSG_DELIMITER   = "]"

	START_BET_DELIMITER  = "{"
	END_BET_DELIMITER    = "}"
	BET_BATCH_SEPARATOR  = ";"
	BET_FIELDS_SEPARATOR = ","
)

// ============================= ENCODE ============================== //

func encodeMessage(messageType string, encodedPayload string) string {
	encodedMessage := messageType
	encodedMessage += START_MSG_DELIMITER
	encodedMessage += encodedPayload
	encodedMessage += END_MSG_DELIMITER
	return encodedMessage
}

func encodeField(fieldName string, fieldValue string) string {
	return fmt.Sprintf(`"%s":"%s"`, fieldName, fieldValue)
}

func EncodeBet(bet *Bet) string {
	encodedBet := START_BET_DELIMITER
	encodedBet += encodeField("agency", bet.Agency) + BET_FIELDS_SEPARATOR
	encodedBet += encodeField("first_name", bet.FirstName) + BET_FIELDS_SEPARATOR
	encodedBet += encodeField("last_name", bet.LastName) + BET_FIELDS_SEPARATOR
	encodedBet += encodeField("document", bet.Document) + BET_FIELDS_SEPARATOR
	encodedBet += encodeField("birthdate", bet.Birthdate) + BET_FIELDS_SEPARATOR
	encodedBet += encodeField("number", bet.Number)
	encodedBet += END_BET_DELIMITER
	return encodedBet
}

func EncodeBetMessage(bet *Bet) string {
	encodedPayload := ""
	encodedPayload += EncodeBet(bet)
	return encodeMessage(BET_MSG_TYPE, encodedPayload)
}

func EncodeAckMessage(message string) string {
	return encodeMessage(ACK_MSG_TYPE, message)
}
