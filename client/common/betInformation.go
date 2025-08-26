package common

import "fmt"

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
