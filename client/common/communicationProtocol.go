package common

const (
	DELIMITER = ']'

	BET_MSG_HEADER = "BET["
	ACK_MSG_HEADER = "ACK["
)

func EncodeBetMessage(bet *Bet) string {
	return BET_MSG_HEADER + bet.AsString() + string(DELIMITER)
}

func EncodeAckMessage(message string) string {
	return ACK_MSG_HEADER + message + string(DELIMITER)
}
