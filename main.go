package main

import (
	"fmt"
)

type Price struct {
	Unix, Open, Close, High, Low int
}

type Stock struct {
	Name   string
	Code   int
	Prices []Price
}
type Text struct {
	Text string
	Unix int
}
type User struct {
	Id     int
	Screen string
	Texts  []Text
}

func main() {
	SaveStockNames("stocknames.csv")
	return
	users := []User{
		User{
			Screen: "h2bl0cker_",
		},
	}
	fmt.Println()
	users = append(users, GetMatchedUsers(50, users, []string{"株", "投資"}, []string{})...)
	UpdateTexts(users)
}
