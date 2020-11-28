package main

import (
	"fmt"
	"time"
)

func main() {
	time.Local, _ = time.LoadLocation("Asia/Tokyo")
	Credentialize("service.json")
	Handle("/market/update", func(w Response, r Request) {
		UpdateMarket()
	})
	Handle("/twitter/update", func(w Response, r Request) {
		UpdatePrediction()
	})
	Handle("/", func(w Response, r Request) {
		markets := make([]Market, 0, 1)
		TableGetAll(NewQuery("MARKET").Limit(cap(markets)).Order("-Born"), &markets)
		predicts := make([]Predict, 0, 1)
		TableGetAll(NewQuery("PREDICT").Limit(cap(predicts)).Order("-Born"), &predicts)
		if len(predicts)==0||len(markets)==0{
			fmt.Fprintln(w,"error")
		}else{
			cp,up:=Possibility{},Possibility{}
			ToData(&cp,predicts[0].CodesPossibility)
			ToData(&up,predicts[0].UsersPossibility)
			fmt.Fprintln(w,"<a href='/twitter/update'>update</a>")
			fmt.Fprintln(w,"<hr>")
			for i,v:=range predicts[0].Codes{
				fmt.Fprintf(w,"<p>#%v <a href='#%v'>%v</a> %v</p>",i,v,v,cp[v])
			}
			fmt.Fprintln(w,"<hr>")
			for i,v:=range predicts[0].Users{
				fmt.Fprintf(w,"<p><a href=https://twitter.com/intent/user?user_id=%v>#%v %v</a></p>",v.Id,i,up[v.Id])
			}
			fmt.Fprintln(w,"<hr>")
			fmt.Fprintf(w,"<p>%v %v</p>",markets[0].Born,len(markets[0].Prices))
			for _,v:=range markets[0].Prices{
				fmt.Fprintf(w,"<p id=%v>#%v %v %v %v %v</p>",v.Code,v.Code, v.FullName,v.Name,v.Open,v.Diff)
			}
		}
	})
	if false {
		UpdatePrediction()
	} else {
		Listen()
	}
}
func UpdateMarket() {
	if TableCount(NewQuery("MARKET").Filter("Born>", time.Now().Add(-12*time.Hour))) == 0 {
		prices := FetchStock()
		if prices != nil {
			TablePut(NewKey("MARKET"), &Market{
				Born:   time.Now(),
				Prices: prices,
			})
		}
	}
}
func UpdatePrediction() {
	markets := make([]Market, 0, 3)
	TableGetAll(NewQuery("MARKET").Limit(cap(markets)).Order("-Born"), &markets)
	predicts := make([]Predict, 0, 1)
	TableGetAll(NewQuery("PREDICT").Limit(cap(predicts)).Order("-Born"), &predicts)
	if len(predicts)==0 {
		predicts=append(predicts,Predict{
			Users: []User{
				{
					Id: SeedUserId,
				},
			},
		})
	}
	predict := Prediction(predicts[0].Users, markets)
	key := predicts[0].Self
	if predict.Born != predicts[0].Born {
		key = NewKey("PREDICT")
	}
	TablePut(key, &predict)
}
func WriteResponse(w Response, params interface{}) {
	WriteTemplate(w, params, nil, "app.html")
}
