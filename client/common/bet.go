package common

import "fmt"

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
	// We set a limit of 512 bytes to be sure, considering header and each field name too
	MAX_BYTES_BET = KiB / 2

	MAX_BYTES_PER_CHUNK = 8 * KiB

	// We will assume that should be at least MAX_BYTES_BET bytes free into the chunck
	// to read a new Bet from csv file
)

type Bet struct {
	agency    string
	number    string
	firstName string
	lastName  string
	document  string
	birthdate string
}

func NewBet(agency string, firstName string, lastName string, document string, birthdate string, number string) *Bet {
	return &Bet{
		agency:    agency,
		firstName: firstName,
		lastName:  lastName,
		document:  document,
		birthdate: birthdate,
		number:    number,
	}
}

func (bet *Bet) AsString() string {
	return fmt.Sprintf(
		`{"agency":"%s","first_name":"%s","last_name":"%s","document":"%s","birthdate":"%s","number":"%s"}`,
		bet.agency,
		bet.firstName,
		bet.lastName,
		bet.document,
		bet.birthdate,
		bet.number,
	)
}

func (bet *Bet) LengthAsString() int {
	return len(bet.AsString())
}
