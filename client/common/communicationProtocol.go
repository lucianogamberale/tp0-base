package common

import (
	"fmt"
	"strings"
)

const (
	MESSAGE_TYPE_LENGTH = 3

	ASK_FOR_WINNERS_MSG_TYPE = "ASK"
	BET_MSG_TYPE             = "BET"
	NO_MORE_BETS_MSG_TYPE    = "NMB"
	ACK_MSG_TYPE             = "ACK"
	WAIT_MSG_TYPE            = "WIT"
	WINNERS_MSG_TYPE         = "WIN"

	START_MSG_DELIMITER = "["
	END_MSG_DELIMITER   = "]"

	START_BET_DELIMITER  = "{"
	END_BET_DELIMITER    = "}"
	BET_BATCH_SEPARATOR  = ";"
	BET_FIELDS_SEPARATOR = ","

	WINNERS_SEPARATOR = ","
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

func EncodeBetBatchMessage(betBatch []*Bet) string {
	encodedPayload := ""
	for i, bet := range betBatch {
		encodedPayload += EncodeBet(bet)
		if i < len(betBatch)-1 {
			encodedPayload += BET_BATCH_SEPARATOR
		}
	}
	return encodeMessage(BET_MSG_TYPE, encodedPayload)
}

func EncodeAckMessage(message string) string {
	return encodeMessage(ACK_MSG_TYPE, message)
}

func EncodeNoMoreBetsMessage(agency string) string {
	encodedPayload := encodeField("agency", agency)
	return encodeMessage(NO_MORE_BETS_MSG_TYPE, encodedPayload)
}

func EncodeAskForWinnersMessage(agency string) string {
	encodedPayload := encodeField("agency", agency)
	return encodeMessage(ASK_FOR_WINNERS_MSG_TYPE, encodedPayload)
}

// ============================= DECODE ============================== //

func DecodeMessageType(message string) (string, error) {
	if len(message) < MESSAGE_TYPE_LENGTH {
		return "", fmt.Errorf("message too short to contain message type")
	}
	return message[0:MESSAGE_TYPE_LENGTH], nil
}

func assertMessageFormat(message string, expectedMessageType string) error {
	receivedMessageType, err := DecodeMessageType(message)
	if err != nil {
		return err
	}

	if receivedMessageType != expectedMessageType {
		return fmt.Errorf("unexpected message type: expected %s but received %s", expectedMessageType, receivedMessageType)
	}

	if !strings.HasPrefix(message, expectedMessageType+START_MSG_DELIMITER) || !strings.HasSuffix(message, END_MSG_DELIMITER) {
		return fmt.Errorf("unexpected message format")
	}
	return nil
}

func getMessagePayload(message string) string {
	// remove message type
	payload := message[MESSAGE_TYPE_LENGTH:]

	// remove message delimiters
	payload = payload[len(START_MSG_DELIMITER) : len(payload)-len(END_MSG_DELIMITER)]

	return payload
}

func DecodeWinnersMessage(message string) ([]string, error) {
	// INPUT: WIN["12135000","87654321", ...]
	err := assertMessageFormat(message, WINNERS_MSG_TYPE)
	if err != nil {
		return nil, err
	}

	payload := getMessagePayload(message)
	if payload == "" {
		return []string{}, nil
	}

	winners := strings.Split(payload, WINNERS_SEPARATOR)
	for i := range winners {
		winners[i] = strings.Trim(winners[i], `"`)
	}

	return winners, nil
}
