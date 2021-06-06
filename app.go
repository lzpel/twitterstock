package main

import (
	"fmt"
	"time"
)

func main() {
	time.Local, _ = time.LoadLocation("Asia/Tokyo")
	Credentialize("service.json")
	TestTwitter()
	return
	UpdatePrediction(false,true,1)
	Handle("/market/update", func(w Response, r Request) {
		UpdateMarket()
	})
	Handle("/twitter/update", func(w Response, r Request) {
		UpdatePrediction(true,true,1)
		fmt.Fprintln(w,"<a href='/'>back</a>")
	})
	Handle("/", func(w Response, r Request) {
		if ServeFile(w,r,r.URL.Path,true){
			return
		}
		markets := make([]Market, 0, 1)
		TableGetAll(NewQuery("MARKET").Limit(cap(markets)).Order("-Born"), &markets)
		predicts := make([]Predict, 0, 1)
		TableGetAll(NewQuery("PREDICT").Limit(cap(predicts)).Order("-Born"), &predicts)
		WriteTemplate(w,map[string]interface{}{
			"Market":markets[0],
			"Predict":predicts[0],
		}, nil,"index.html")
	})
	if false {
		TestTwitter()
		//UpdatePrediction(true,false,1)
	} else {
		Listen()
	}
}
func UpdateMarket() {
	if market := FetchMarket(); market != nil {
		TablePut(NewKey("MARKET"), market)
	}
}
func ClearPrediction(){
	for true{
		keys:=TableGetAll(NewQuery("PREDICT").Limit(1000).Order("-Born").KeysOnly(), nil)
		if keys==nil||len(keys)==0{
			break
		}
		TableDeleteAll(keys)
		time.Sleep(time.Second)
	}
}
func UpdatePrediction(put bool,useCache bool,count int) Predict{
	markets := make([]Market, 0, 5)
	TableGetAll(NewQuery("MARKET").Limit(cap(markets)).Order("-Born"), &markets)
	predicts := make([]Predict, 0, 1)
	TableGetAll(NewQuery("PREDICT").Limit(cap(predicts)).Order("-Born"), &predicts)
	if len(predicts)==0{
		predicts=append(predicts,Predict{
			Born:Daily(time.Now()),
			Users:[]User{
				User{
					Screen:"masahrhz",
					Id:2852732372,
					Name:"まさ",
				},
			},
			Prices:[]Price{},
		})
	}
	predict:=predicts[0]
	for i:=0;i<count;i++{
		predict = Prediction(predict.Users, markets,useCache)
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
