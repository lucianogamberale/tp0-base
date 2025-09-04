package common

const (
	KiB = 1024

	// MAX_BYTES_BET defines the upper bound (in bytes) for the serialized
	// representation of a single Bet. While the raw data of a bet is ~135 bytes
	// (agency + first_name + last_name + document + birthdate + number),
	// we set a conservative limit of 256 bytes (1/4 KiB).
	//
	// This margin accounts for:
	//   - field names (e.g. "first_name=")
	//   - delimiters
	//   - protocol header overhead
	//
	// By enforcing this size, we can:
	//   - validate message integrity
	//   - prevent malformed/oversized inputs
	//   - simplify chunk allocation when streaming bets (e.g. from CSV)
	//
	// Assumption: a new bet will only be read if there are at least
	// MAX_BYTES_BET free bytes available in the buffer.
	MAX_BYTES_BET = KiB / 4
)

type Bet struct {
	Agency    string
	Number    string
	FirstName string
	LastName  string
	Document  string
	Birthdate string
}

func NewBet(agency string, firstName string, lastName string, document string, birthdate string, number string) *Bet {
	return &Bet{
		Agency:    agency,
		FirstName: firstName,
		LastName:  lastName,
		Document:  document,
		Birthdate: birthdate,
		Number:    number,
	}
}

func (bet *Bet) LengthWhenEncoded() int {
	return len(EncodeBet(bet))
}
