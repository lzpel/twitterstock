package main

type Price struct {
	Unix, Open, Close, High, Low int
}

type Stock struct {
	Name   string
	Code   int
	Market rune
	Prices []Price
}
type Text struct {
	Text string
	Unix int
}
type User struct {
	Id     int
	Screen, Name string
	Texts  []Text
}

func main(){
	FetchStocks("stocksall.csv")
}
func main2() {
	users := []User{
		User{
			Name:"ぱちょ@株とDVC",
			Id:1086060182860292096,
			Screen: "h2bl0cker_",
		},
	}
	users = append(users, FetchUsers(50, users, []string{"株", "投資"}, []string{})...)
	SaveUsers("users.csv",users)
	UpdateTexts(users)
}
