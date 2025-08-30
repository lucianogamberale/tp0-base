package common

const (
	BET_MSG_TYPE = "BET"
	ACK_MSG_TYPE = "ACK"

	START_DELIMITER = "["
	END_DELIMITER   = "]"

	BET_SEPARATOR = ","
)

func EncodeBetMessage(bet *Bet) string {
	return BET_MSG_TYPE + START_DELIMITER + bet.AsString() + END_DELIMITER
}

func EncodeBetBatchMessage(betBatch []*Bet) string {
	message := BET_MSG_TYPE + START_DELIMITER

	for i, bet := range betBatch {
		message += bet.AsString()
		if i < len(betBatch)-1 {
			message += BET_SEPARATOR
		}
	}

	message += END_DELIMITER
	return message
}

func EncodeAckMessage(message string) string {
	return ACK_MSG_TYPE + START_DELIMITER + message + END_DELIMITER
}
