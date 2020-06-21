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
	"strings"
	"time"
)

func SaveStockNames(csvpath string) {
	file, e := os.OpenFile(csvpath, os.O_WRONLY|os.O_CREATE, 0666)
	if e != nil {
		log.Fatal(e)
	}
	defer file.Close()
	for page := 0; page < 82; {
		time.Sleep(time.Second * 5)
		url := fmt.Sprintf("http://portal.morningstarjp.com/StockInfo/sec/list?page=%d", page)
		if code, body := Curl(url,true); code == 200 {
			ex := regexp.MustCompile(`(\d+)"([^/]+)/a/tdtdclass="tac"東証１部`)
			matches := ex.FindAllStringSubmatch(body, -1)
			for _, match := range matches {
				fmt.Fprintf(file,"%s,%s\n", match[1], match[2])
			}
			fmt.Println(page,len(matches))
			page++
		}
	}
}
func GetStocks(csvpath, include string, colname, colcode int) []Stock {
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
	returnStocks := []Stock{}
	for _, row := range rows {
		flag := false
		for _, m := range row {
			flag = flag || strings.Contains(m, include)
		}
		if flag {
			stock := Stock{}
			if stock.Code, e = strconv.Atoi(row[colcode]); e == nil {
				stock.Prices = GetPrices(stock.Code)
				returnStocks = append(returnStocks, stock)
			} else {
				log.Println("WARNING", e)
			}
		}
	}
	return returnStocks
}

func GetPrices(code int) (outs []Price) {
	url := fmt.Sprintf("https://stocks.finance.yahoo.co.jp/stocks/history/?code=%4d.T", code)
	if code, body := Curl(url,false); code == 200 {
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
func Curl(url string, shiftjis bool) (int, string) {
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
