package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

func Fetch(url string,) (int, string) {
	if w, e := http.Get(url);e==nil {
		defer w.Body.Close()
		bytes, _ := ioutil.ReadAll(w.Body)
		body := regexp.MustCompile(`(?m)[\s,"]`).ReplaceAllString(string(bytes), "")
		return w.StatusCode, body
	}
	return 0,""
}

func ToInt(s string) int{
	r,_:=strconv.Atoi(s)
	return r
}

func FetchStock() []Price{
	timetoday:=time.Now()
	result:=make([]Price,0,500)
	for page := 1; page <= 10; page++{
		url := fmt.Sprintf("https://www.nikkei.com/markets/kabu/nidxprice/?StockIndex=N500&Gcode=00&hm=%d", page)
		if code, body := Fetch(url); code == 200 {
			//fmt.Println(body)
			timematch:=regexp.MustCompile(`更新:(\d+)/(\d+)/(\d+)`).FindStringSubmatch(body)
			if(timetoday.Day()!=ToInt(timematch[3])){
				return nil
			}
			matches := regexp.MustCompile(`<trclass=tr2>.*?title=(.*?)の株価情報>(\d+).*?株価情報>(.*?)</a></td>.*?(\d+)<br>.*?(\d+)<br>.*?(\d+)<br>.*?(\d+)</span>.*?([－＋±])(\d+).*?</tr>`).FindAllStringSubmatch(body, -1)
			for _, match := range matches {
				fmt.Println(match)
				p:=Price{
					FullName: match[1],
					Code:  ToInt(match[2]),
					Name:  match[3],
					Open:  ToInt(match[4]),
					High:  ToInt(match[5]),
					Low:   ToInt(match[6]),
					Close: ToInt(match[7]),
					Diff:  ToInt(match[9])*map[string]int{"－":-1, "±":0, "＋":+1,}[match[8]],
				}
				result=append(result, p)
			}
		}else{
			return nil
		}
		time.Sleep(time.Millisecond * 500)
	}
	return result
}