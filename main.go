package main

import (
	"encoding/csv"
	"fmt"
	"github.com/ChimeraCoder/anaconda"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

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
	Id           int
	Screen, Name string
	Texts        []Text
}

func main() {
	//FetchStocks("stocks.csv")
	LoadStocks("stocks.csv")
}
func main2() {
	users := []User{
		User{
			Name:   "ぱちょ@株とDVC",
			Id:     1086060182860292096,
			Screen: "h2bl0cker_",
		},
	}
	users = append(users, FetchUsers(50, users, []string{"株", "投資"}, []string{})...)
	SaveUsers("users.csv", users)
	UpdateTexts(users)
}

func FetchStocks(csvpath string) {
	file, e := os.OpenFile(csvpath, os.O_WRONLY|os.O_CREATE, 0666)
	if e != nil {
		log.Fatal(e)
	}
	defer file.Close()
	for page := 0; page < 82; {
		time.Sleep(time.Second * 5)
		url := fmt.Sprintf("http://portal.morningstarjp.com/StockInfo/sec/list?page=%d", page)
		if code, body := Fetch(url, true); code == 200 {
			ex := regexp.MustCompile(`(\d+)"([^/]+)/a/tdtdclass="tac"(東証１部|東証２部|マザーズ|ＪＡＳＤＡＱ)`)
			matches := ex.FindAllStringSubmatch(body, -1)
			for _, match := range matches {
				fmt.Fprintf(file, "%s,%s,%s\n",
					match[1],
					map[string]string{
						"東証１部":   "1",
						"東証２部":   "2",
						"マザーズ":   "m",
						"ＪＡＳＤＡＱ": "j",
					}[match[3]],
					match[2],
				)
			}
			fmt.Println(page, len(matches))
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
	returnStocks := make([]Stock, 0, len(rows))
	for _, row := range rows {
		if len(row) == 3 {
			code, _ := strconv.Atoi(row[0])
			returnStocks = append(returnStocks, Stock{
				Code:   code,
				Market: rune(row[1][0]),
				Name:   row[2],
			})
		}
	}
	return returnStocks
}

func FetchPrices(code int) (outs []Price) {
	url := fmt.Sprintf("https://stocks.finance.yahoo.co.jp/stocks/history/?code=%4d.T", code)
	if code, body := Fetch(url, false); code == 200 {
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
	if e == nil {
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

func NewApi() *anaconda.TwitterApi {
	return anaconda.NewTwitterApiWithCredentials("828661472-lMGvALleFfL15jIkFhCgvhhph4ZpMKYlI573cI7a", "Kqch8IsgLBXSbhYvUVzDYSttA4LVl0pcUo55GSVA3CEMT", "mCPWWS6PazLgF36jkUnG4oPIT", "deqbZQubFuYWTWznQV2WfNL93sXoOSyZZsnkefJABb1taXwhYP")
}

func Min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func ShuffleUsers(returnUsers []User) []User {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(returnUsers), func(i, j int) {
		returnUsers[i], returnUsers[j] = returnUsers[j], returnUsers[i]
	})
	return returnUsers
}
func SaveUsers(csvpath string, users []User) {
	file, e := os.OpenFile(csvpath, os.O_WRONLY|os.O_CREATE, 0666)
	if e != nil {
		log.Fatal(e)
	}
	defer file.Close()
	for _, user := range users {
		fmt.Fprintf(file, "%d,%s,%s\n", user.Id, user.Screen, user.Name)
	}
}
func LoadUsers(csvpath string) []User {
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
	returnUsers := make([]User, 0, len(rows))
	for _, row := range rows {
		if len(row) == 3 {
			code, _ := strconv.Atoi(row[0])
			returnUsers = append(returnUsers, User{
				Name:   row[2],
				Screen: row[1],
				Id:     code,
			})
		}
	}
	return returnUsers
}
func FetchUsers(count int, seedusers []User, include, exclude []string) []User {
	api := NewApi()
	returnUsers := []User{}
	seedusers = ShuffleUsers(seedusers)
	for i := 0; i < Min(5, len(seedusers)); i++ {
		user := seedusers[i]
		v := url.Values{}
		v.Set("count", strconv.Itoa(50))
		v.Set("user_id", strconv.Itoa(user.Id))
		//v.Set("screen_name", user.Screen)
		if res, e := api.GetFriendsList(v); e == nil {
			for _, n := range res.Users {
				match, description := false, n.Name+n.Description
				for _, w := range include {
					match = match || strings.Contains(description, w)
				}
				for _, w := range exclude {
					match = match && !strings.Contains(description, w)
				}
				log.Println(match, n.ScreenName, description)
				if match {
					returnUsers = append(returnUsers, User{
						Screen: n.ScreenName,
						Name:   n.Name,
						Id:     int(n.Id),
					})
				}
			}
		}
	}
	if count > 0 {
		returnUsers = ShuffleUsers(returnUsers)
		returnUsers = returnUsers[:Min(len(returnUsers), count)]
	}
	return returnUsers
}
func UpdateTexts(users []User) {
	api := NewApi()
	for _, user := range users {
		v := url.Values{}
		if user.Id > 0 {
			v.Set("user_id", strconv.Itoa(user.Id))
		} else {
			v.Set("screen_name", user.Screen)
		}
		api.GetUserTimeline(v)
	}
}
