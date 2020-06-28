package main

import (
	"encoding/csv"
	"fmt"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
)

func FetchStocks(csvpath string) {
	file, e := os.OpenFile(csvpath, os.O_WRONLY|os.O_CREATE, 0666)
	if e != nil {
		log.Fatal(e)
	}
	defer file.Close()
	for page := 0; page < 82; {
		time.Sleep(time.Second * 5)
		url := fmt.Sprintf("http://portal.morningstarjp.com/StockInfo/sec/list?page=%d", page)
		if code, body := Fetch(url,true); code == 200 {
			ex := regexp.MustCompile(`(\d+)"([^/]+)/a/tdtdclass="tac"(東証１部|東証２部|マザーズ|ＪＡＳＤＡＱ)`)
			matches := ex.FindAllStringSubmatch(body, -1)
			for _, match := range matches {
				fmt.Fprintf(file,"%s,%s,%s\n",
					match[1],
					map[string]string{
						"東証１部":"1",
						"東証２部":"2",
						"マザーズ":"m",
						"ＪＡＳＤＡＱ":"j",
					}[match[3]],
					match[2],
				)
			}
			fmt.Println(page,len(matches))
			page++
		}
	}
}
func LoadStocks(csvpath string) []Stock {
	//ファイルを開く
	f, e := os.Open(csvpath)
	if e != nil {
		log.Fatal(e)
	}
	defer f.Close()
	//ファイルを読み込む
	rows, e := csv.NewReader(f).ReadAll()
	if e != nil {
		log.Fatal(e)
	}
	returnStocks := make([]Stock,0,len(rows))
	for _, row := range rows {
		if len(row)==3{
			code,_ := strconv.Atoi(row[0])
			returnStocks = append(returnStocks, Stock{
				Name: row[1],
				Market: rune(row[2][0]),
				Code: code,
			})
		}
	}
	return returnStocks
}

func FetchPrices(code int) (outs []Price) {
	url := fmt.Sprintf("https://stocks.finance.yahoo.co.jp/stocks/history/?code=%4d.T", code)
	if code, body := Fetch(url,false); code == 200 {
		matches := regexp.MustCompile(`trtd(\d+年\d+月\d+日)/tdtd(\d+)/tdtd(\d+)/tdtd(\d+)/tdtd(\d+)/tdtd(\d+)/tdtd(\d+)/td/tr`).FindAllStringSubmatch(body, -1)
		outs = make([]Price, len(matches))
		for i := 0; i < len(matches); i++ {
			p := &outs[i]
			t, _ := time.Parse("2006年1月2日", matches[i][1])
			p.Unix = int(t.Unix())
			p.Open, _ = strconv.Atoi(matches[i][2])
			p.High, _ = strconv.Atoi(matches[i][3])
			p.Low, _ = strconv.Atoi(matches[i][4])
			p.Close, _ = strconv.Atoi(matches[i][5])
		}
	}
	if outs == nil || len(outs) < 5 {
		log.Println("WARNING", code, outs)
	}
	return outs
}
func Fetch(url string, shiftjis bool) (int, string) {
	var r io.Reader
	w, e := http.Get(url)
	if e==nil{
		defer w.Body.Close()
	}
	if shiftjis {
		r = transform.NewReader(w.Body, japanese.ShiftJIS.NewDecoder())
	} else {
		r = w.Body
	}
	bytes, _ := ioutil.ReadAll(r)
	body := regexp.MustCompile(`(?m)[\s,<>]`).ReplaceAllString(string(bytes), "")
	return w.StatusCode, body
}
