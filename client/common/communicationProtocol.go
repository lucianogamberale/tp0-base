package common

const (
	DELIMITER = ']'

	BET_MSG_HEADER = "BET["
	ACK_MSG_HEADER = "ACK["
)

func BetMessageFor(bet *Bet) string {
	return BET_MSG_HEADER + bet.AsString() + string(DELIMITER)
}

func AckMessage(ack string) string {
	return ACK_MSG_HEADER + ack + string(DELIMITER)
}
