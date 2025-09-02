package common

const (
	KiB = 1024

	// Maximum size of a Bet in bytes:
	//
	// agency (5 bytes) +
	// first_name (50 bytes) +
	// last_name (50 bytes) +
	// document (10 bytes) +
	// birthdate (10 bytes) +
	// number (10 bytes) = 135 bytes
	//
	// We set a limit of 256 bytes to be sure, considering header and each field name too
	MAX_BYTES_BET = KiB / 4

	// We will assume that should be at least MAX_BYTES_BET bytes free into the chunk
	// to read a new Bet from csv file
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
