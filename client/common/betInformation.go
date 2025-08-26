package common

import "fmt"

type BetInformation struct {
	agencyId        string
	betId           string
	playerName      string
	playerLastname  string
	playerDni       string
	playerBirthDate string
}

func NewBetInformation(agencyId string, betId string, playerName string, playerLastname string, playerDni string, playerBirthDate string) *BetInformation {
	return &BetInformation{
		agencyId:        agencyId,
		betId:           betId,
		playerName:      playerName,
		playerLastname:  playerLastname,
		playerDni:       playerDni,
		playerBirthDate: playerBirthDate,
	}
}

func (betInformation *BetInformation) AsString() string {
	return fmt.Sprintf(
		`{"agencyId":"%s","betId":"%s","playerName":"%s","playerLastname":"%s","playerDni":"%s","playerBirthDate":"%s"}`,
		betInformation.agencyId,
		betInformation.betId,
		betInformation.playerName,
		betInformation.playerLastname,
		betInformation.playerDni,
		betInformation.playerBirthDate,
	)
}
