package main

import (
	"fmt"
	"time"
)

func main() {
	time.Local, _ = time.LoadLocation("Asia/Tokyo")
	Credentialize("service.json")
	Handle("/market", func(w Response, r Request) {
		UpdateMarket()
	})
	Handle("/twitter", func(w Response, r Request) {
		UpdatePrediction(true, true, 1)
		fmt.Fprintln(w, "<a href='/'>back</a>")
	})
	Handle("/", func(w Response, r Request) {
		if ServeFile(w, r, r.URL.Path, true) {
			return
		}
		markets := make([]Market, 0, 1)
		TableGetAll(NewQuery("MARKET").Limit(cap(markets)).Order("-Born"), &markets)
		predicts := make([]Predict, 0, 1)
		TableGetAll(NewQuery("PREDICT").Limit(cap(predicts)).Order("-Born"), &predicts)
		WriteTemplate(w, map[string]interface{}{
			"Market":  markets,
			"Predict": predicts,
		}, Dict{
			"Local": func(t time.Time) string {
				return t.In(time.Local).Format("2006-01-02 15:04")
			},
			"Coefficient": func(t float64) string {
				return fmt.Sprintf("%+.5f", t)
			},
			"Percent": func(t float64) string {
				return fmt.Sprintf("%+.2f%%", t*100)
			},
		}, "index.html")
	})
	if false {
		ModifyPrediction()
		//TestTwitter()
		//UpdatePrediction(true, true, 1)
	} else {
		Listen()
	}
}
func (pr *Predict) MentionPrice(p *Price) map[string]int64 {
	v := map[string]int64{}
	for k, _ := range pr.Users {
		u:=pr.Users[k]
		m := u.Mention
		for i := 0; i < len(m); i += 2 {
			if (m[i]>>16) == pr.Born.Unix() && int(m[i]&0xffff) == p.Code {
				v[u.Screen]=m[i+1]
			}
		}
	}
	return v
}

func (pr *Predict) MentionUser(u *User) map[int64]int {
	v := map[int64]int{}
	m := u.Mention
	for i := 0; i < len(m); i += 2 {
		if (m[i]>>16) == pr.Born.Unix() && int(m[i]&0xffff)!=0{
			v[m[i+1]]=int(m[i]&0xffff)
		}
	}
	return v
}

func UpdateMarket() {
	if market := FetchMarket(); market != nil {
		TablePut(NewKey("MARKET"), market)
	}
}
func UpdatePrediction(put bool, useCache bool, count int) Predict {
	markets := make([]Market, 0, MarketDays)
	TableGetAll(NewQuery("MARKET").Limit(cap(markets)).Order("-Born"), &markets)
	predicts := make([]Predict, 0, 1)
	TableGetAll(NewQuery("PREDICT").Limit(cap(predicts)).Order("-Born"), &predicts)
	if len(predicts) == 0 {
		predicts = []Predict{
			{
				Users: []User{
					User{
						Screen: "masahrhz",
						Id:     2852732372,
						Name:   "まさ",
					},
				},
				Prices: []Price{},
			},
		}
	}
	var predict Predict
	for i := 0; i < count; i++ {
		predict = Prediction(predicts[0].Users, markets, markets[0].Prices, time.Now(), useCache)
		if put {
			TablePut(predict.Key(), &predict)
		}
		predicts[0] = predict
	}
	return predict
}

func ModifyPrediction() {
	if false{
		predicts := make([]Predict, 0, 1)
		TableGetAll(NewQuery("PREDICT").Limit(cap(predicts)).Order("-Born"), &predicts)
		predict:=predicts[0]
		users:=predict.Users
		predict.Users=make([]User,0,len(users))
		for _,v:=range users{
			if IsValidUser(&v){
				predict.Users=append(predict.Users,v)
				fmt.Println(true, v.Screen, v.Description)
			}else{
				fmt.Println(false, v.Screen ,v.Description)
			}
		}
		fmt.Println(len(predict.Users))
		TablePut(predict.Key(),&predict)
	}
	if false{
		for true {
			keys := TableGetAll(NewQuery("PREDICT").Limit(1000).Order("-Born").KeysOnly(), nil)
			if keys == nil || len(keys) == 0 {
				break
			}
			TableDeleteAll(keys)
			time.Sleep(time.Second)
		}
	}
}
