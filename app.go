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
		UpdatePrediction(true,1)
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
	if true {
		p:=UpdatePrediction(true,10)
		_=p
	} else {
		Listen()
	}
}
func UpdateMarket() {
	if market := FetchMarket(); market != nil {
		TablePut(NewKey("MARKET"), market)
	}
}
func UpdatePrediction(put bool,count int) Predict{
	markets := make([]Market, 0, 3)
	TableGetAll(NewQuery("MARKET").Limit(cap(markets)).Order("-Born"), &markets)
	predicts := make([]Predict, 0, 1)
	TableGetAll(NewQuery("PREDICT").Limit(cap(predicts)).Order("-Born"), &predicts)
	predict:=predicts[0]
	for i:=0;i<count;i++{
		predict = Prediction(predict.Users, markets)
		if put{
			key := predicts[0].Self
			if predict.Born != predicts[0].Born {
				key = NewKey("PREDICT")
			}
			TablePut(key, &predict)
		}
	}
	return predict
}
func WriteResponse(w Response, params interface{}) {
	WriteTemplate(w, params, nil, "app.html")
}
