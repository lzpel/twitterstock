package main

import "time"

func main() {
	Handle("/market/update", func(w Response, r Request) {
		count := TableCount(NewQuery("MARKET").Filter("Born>", time.Now().Add(-12*time.Hour)))
		if count == 0 {
			TablePut("MARKET", &Market{
				Born:   time.Now(),
				Prices: FetchStock(),
			})
		}
	})
	Credentialize("service.json")
	Listen()
}
func WriteResponse(w Response, params interface{}) {
	WriteTemplate(w, params, nil, "app.html")
}
