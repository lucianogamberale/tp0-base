package common

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
