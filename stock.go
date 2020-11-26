package main

import (
	"fmt"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

func Fetch(url string, shiftjis bool) (int, string) {
	var r io.Reader
	w, e := http.Get(url)
	if e == nil {
		defer w.Body.Close()
	}
	if shiftjis {
		r = transform.NewReader(w.Body, japanese.ShiftJIS.NewDecoder())
	} else {
		r = w.Body
	}
	bytes, _ := ioutil.ReadAll(r)
	body := regexp.MustCompile(`(?m)[\s,"]`).ReplaceAllString(string(bytes), "")
	return w.StatusCode, body
}

func ToInt(s string) int{
	r,_:=strconv.Atoi(s)
	return r
}

func FetchStock() []Price{
	loc, _ := time.LoadLocation("Asia/Tokyo")
	timetoday:=time.Now().In(loc)
	result:=make([]Price,0,500)
	for page := 1; page <= 10; page++{
		url := fmt.Sprintf("https://www.nikkei.com/markets/kabu/nidxprice/?StockIndex=N500&Gcode=00&hm=%d", page)
		if code, body := Fetch(url, false); code == 200 {
			//fmt.Println(body)
			timematch:=regexp.MustCompile(`更新:(\d+)/(\d+)/(\d+)`).FindStringSubmatch(body)
			if(timetoday.Day()!=ToInt(timematch[3])){
				return nil
			}
			matches := regexp.MustCompile(`<trclass=tr2>.*?株価情報>(\d+).*?株価情報>(.*?)</a></td>.*?(\d+)<br>.*?(\d+)<br>.*?(\d+)<br>.*?(\d+)</span>.*?([－＋±])(\d+).*?</tr>`).FindAllStringSubmatch(body, -1)
			for _, match := range matches {
				fmt.Println(match)
				p:=Price{
					Code:  ToInt(match[1]),
					Name:  match[2],
					Open:  ToInt(match[3]),
					High:  ToInt(match[4]),
					Low:   ToInt(match[5]),
					Close: ToInt(match[6]),
					Diff:  ToInt(match[8])*map[string]int{"－":-1, "±":0, "＋":+1,}[match[7]],
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