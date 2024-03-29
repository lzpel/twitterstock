package main

import (
	"fmt"
	"github.com/ChimeraCoder/anaconda"
	"github.com/sajari/regression"
	"golang.org/x/text/unicode/norm"
	"log"
	"math"
	"math/rand"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

// @fn
// API取得
func NewApi() *anaconda.TwitterApi {
	return anaconda.NewTwitterApiWithCredentials(os.Getenv("ACCESS_TOKEN"), os.Getenv("ACCESS_SECRET"), os.Getenv("CONSUMER_TOKEN"), os.Getenv("CONSUMER_SECRET"))
}

/// @variable
var PredictTweet = []int64{0, 0, 0, 0}
var IgnoreWordsList = []string{}

// @fn
// unixtimeに最も近い時刻のツイートのIDを予測する関数
// 一次近似を仮定しているが十分正確
func PredictTweetTime(id, unix int64) int64 {
	if id != 0 && unix != 0 {
		if PredictTweet[0] < id {
			PredictTweet[0] = id
			PredictTweet[1] = unix
		}
		if PredictTweet[2] < id && (PredictTweet[1]-unix) > 3600*6 {
			PredictTweet[2] = id
			PredictTweet[3] = unix
		}
		return 0
	}
	if PredictTweet[0] == 0 || PredictTweet[2] == 0 || PredictTweet[2] == PredictTweet[0] {
		v := url.Values{}
		v.Set("user_id", "783214")
		v.Set("count", strconv.Itoa(200))
		v.Set("trim_user", "true")
		v.Set("exclude_replies", "false")
		if tweets, e := NewApi().GetUserTimeline(v); e != nil {
			log.Fatalln(e)
		} else {
			for _, v := range tweets {
				PredictTweetTimeUpdate(&v)
			}
		}
	}
	if id == 0 {
		return PredictTweet[2] + int64(float64(unix-PredictTweet[3])*float64(PredictTweet[0]-PredictTweet[2])/float64(PredictTweet[1]-PredictTweet[3]))
	}
	if unix == 0 {
		return PredictTweet[3] + int64(float64(id-PredictTweet[2])/float64(PredictTweet[0]-PredictTweet[2])*float64(PredictTweet[1]-PredictTweet[3]))
	}
	return 0
}

func PredictTweetTimeUpdate(v *anaconda.Tweet) {
	if t, e := v.CreatedAtTime(); e == nil {
		PredictTweetTime(v.Id, t.Unix())
	}
}

// @fn
// 最新の特定時刻に揃えた時刻を返す
func DailyDuration(t time.Time, duration time.Duration) time.Time {
	// 東証の取引時間は現在、午前９時―午後３時で、途中１時間の休憩が入る。
	duration += time.Hour * time.Duration(DeadlineHour)
	t = t.In(time.Local).Add(-duration)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local).Add(+duration)
}

func Daily(t time.Time) time.Time {
	return DailyDuration(t, 0)
}

func Deadline(t time.Time) time.Time {
	return Daily(t)
}

func FormatString(s string) string {
	return norm.NFKC.String(s)
}

// @fn
// 銘柄名の有効判定
func IsValidName(s string) bool {
	score := 0
	for _, c := range s {
		if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			score += 75 // 300/4==75
		} else if unicode.In(c, unicode.Hiragana) || unicode.In(c, unicode.Katakana) {
			score += 75
		} else {
			score += 100
		}
	}
	if strings.Contains(ExcludePriceWords, s) {
		score = 0
	}
	return score >= 300
}

func IsValidUser(u *User) bool {
	if len(IgnoreWordsList) == 0 {
		IgnoreWordsList = strings.Split(IncludeUserWords, ",")
	}
	for _, v := range IgnoreWordsList {
		if strings.Contains(u.Description, v) && len(v) != 0 {
			return true
		}
	}
	return false
}

func HasReference(text string, p *Price) bool {
	tw, ns, nl := FormatString(text), FormatString(p.Name), FormatString(p.FullName)
	if (strings.Contains(tw, ns) && IsValidName(ns)) || (strings.Contains(tw, nl) && IsValidName(nl)) {
		return true
	}
	return false
}

func MentionUpdate(user *User , times map[int64][]Price) map[int64]Mention{
	//与えられた時刻列~24時間前のツイートを補完する
	//Dailyを呼び出してはならない
	mentions := map[int64]Mention{}
	for i := 0; i < len(user.Mention); i += 2 {
		m := Mention{
			Hash: user.Mention[i],
			Id:   user.Mention[i+1],
		}
		mentions[m.Hash]=m
	}
	var minTime, maxTime time.Time
	for k, _ := range times {
		start := Mention{}
		start.Set(k, 0, 0)
		if _, ok := mentions[start.Hash]; ok == false {
			startTime := start.Time()
			if maxTime.Unix() < 0 || maxTime.Unix() < startTime.Unix() {
				maxTime = startTime
			}
			startTime=startTime.Add(-24 * time.Hour)
			if minTime.Unix() < 0 || minTime.Unix() > startTime.Unix() {
				minTime = startTime
			}
			mentions[start.Hash] = start
		}
	}
	if minTime.Unix() == maxTime.Unix() {
		fmt.Println("Skipped", user.Name, user.Screen)
	} else {
		maxID, minID := PredictTweetTime(0, maxTime.Unix()), PredictTweetTime(0, minTime.Unix())
		fmt.Println(maxTime, minTime, maxID, minID)
		for maxID != minID {
			v := url.Values{}
			v.Set("user_id", strconv.Itoa(user.Id))
			v.Set("max_id", strconv.FormatInt(maxID, 10))
			//v.Set("since_id", strconv.FormatInt(minID, 10))
			v.Set("count", strconv.Itoa(200))
			v.Set("exclude_replies", "false")
			v.Set("trim_user", "true")
			if page, err := NewApi().GetUserTimeline(v); err != nil {
				fmt.Printf("no tweets https://twitter.com/%v\n", user.Screen)
				break
			} else if len(page) <= 2 {
				//終了条件1：アカウントの取得可能な最古のツイートを発見した
				break
			} else {
				for _, tw := range page {
					PredictTweetTimeUpdate(&tw)
					if minID > tw.Id {
						//終了条件2：必要分より古いツイートを発見した
						maxID = minID
						break
					} else {
						maxID = tw.Id - 1
					}
					born, _ := tw.CreatedAtTime()
					for k, prices := range times {
						if born.Unix() <= k && k < born.Add(24*time.Hour).Unix() {
							for _, p := range prices {
								if HasReference(tw.FullText, &p) {
									start := Mention{}
									start.Set(born.Unix(), p.Code, tw.Id)
									mentions[start.Hash] = start
								}
							}
						}
					}
				}
			}
		}
		fmt.Println("Done", user.Name, user.Screen)
	}
	return mentions
}

func MentionUser(user *User, markets []Market, mentionMap map[*User][]*Price, mu *sync.Mutex, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	fmt.Println("MentionUser", user.Name, user.Screen)
	Age := time.Now().Add(-CacheAge)
	times := map[int64][]Price{}
	for _, m := range markets {
		//当日の基準時刻を用いることはここで判明する
		times[Deadline(m.Born).Unix()] = m.Prices
	}
	mentions:=MentionUpdate(user, times)
	user.Mention = user.Mention[:0]
	prices := map[*Price]int{}
	for _, v := range mentions {
		if Age.Unix() > v.Time().Unix() {
			continue
		}
		//言及保持継続
		user.Mention = append(user.Mention, v.Hash, v.Id)
		tweet := v.Time()
		//時刻一致で言及マップに追加
		for j, _ := range markets {
			market := &markets[j]
			dead := Deadline(market.Born)
			if v.Code() != 0 && tweet.Unix() <= dead.Unix() && dead.Unix() < tweet.Add(24*time.Hour).Unix() {
				//新規上場銘柄への言及などで過去の銘柄テーブルに無い銘柄を参照した場合無視する。
				for k, _ := range market.Prices {
					if price := &market.Prices[k]; price.Code == v.Code() {
						prices[price] = len(prices)
					}
				}
			}
		}
	}
	if mu != nil {
		mu.Lock()
	}
	for k, _ := range prices {
		mentionMap[user] = append(mentionMap[user], k)
	}
	if mu != nil {
		mu.Unlock()
	}
}
func PredictRegression(usersMap map[*User]int, mentionMap map[*Price][]*User) {
	//予測
	r := new(regression.Regression)
	r.SetObserved("株価の変動率")
	for k, m := range usersMap {
		r.SetVar(m, k.Name)
	}
	predict := map[*Price][]float64{}
	for p, pusers := range mentionMap {
		v, skip := make([]float64, len(usersMap)), true
		for _, n := range pusers {
			if idx, ok := usersMap[n]; ok {
				v[idx] = 1.0
				skip = false
			} else {
				//ここに到達する場合、nは過去の言及が無く未来の言及が有るユーザー、NaN係数を避けるため無視する
			}
		}
		if skip == true {
			//有効な言及者がいない
			continue
		}
		if p.High >= 0 {
			r.Train(regression.DataPoint(p.Target(), v))
		} else {
			predict[p] = v
		}
	}
	if e := r.Run(); e != nil {
		fmt.Println(e)
	} else {
		fmt.Println(r.Formula)
		fmt.Println(r)
	}
	for k, m := range usersMap {
		k.Coefficient = r.Coeff(m + 1)
	}
	for k, m := range predict {
		var e error
		if k.PredictRegression, e = r.Predict(m); e != nil {
			fmt.Println("E: 予測不能")
		}
		fmt.Println(k.Name, k.PredictRegression, m)
	}
}
func PredictBayesian(mentionReverseMap map[*User][]*Price, mentionMap map[*Price][]*User, markets []Market) {
	ga1, ga2 := 0, 0
	for _, m := range markets {
		for _, p := range m.Prices {
			if p.High > 0 {
				if p.Target() > 0 {
					ga1++
				} else {
					ga2++
				}
			}
		}
	}
	for k, v := range mentionMap {
		//初期確率
		if k.High > 0 {
			continue
		}
		pa1, pa2 := 0, 0
		for _, m := range markets {
			for _, p := range m.Prices {
				if p.High > 0 && p.Code == k.Code {
					if p.Target() > 0 {
						pa1++
					} else {
						pa2++
					}
				}
			}
		}
		k.PredictBayesian = float64(pa1) / float64(pa1+pa2)
		fmt.Println(k.FullName, "初期確率", k.PredictBayesian, pa1, pa2)
		//ベイズ更新
		for _, u := range v {
			ua1, ua2 := 0, 0
			for _, p := range mentionReverseMap[u] {
				if p.High > 0 {
					if p.Target() > 0 {
						ua1++
					} else {
						ua2++
					}
				}
			}
			if ua1*ua2 != 0 {
				k.PredictBayesian = (float64(ua1) / float64(ga1)) * k.PredictBayesian / ((float64(ua1)/float64(ga1))*k.PredictBayesian + (float64(ua2)/float64(ga2))*(1-k.PredictBayesian))
			}
			fmt.Println(k.FullName, "更新確率", k.PredictBayesian, ua1, ua2)
		}
	}
}

/// @fn
func Train(users []User, prices []Price, future time.Time, markets []Market) ([]User, []Price, bool) {
	//アドレスを用いて日付x価格と人物のmapを作成しているのでusersとmarketsとPricesは複製してはならない。しかしmarkets[n].Pricesとpredictは別領域である必要がある。
	marketFuture := Market{
		Prices: append([]Price{}, prices...),
		Born:   Daily(future),
	}
	fmt.Println(marketFuture.Born)
	for k, _ := range marketFuture.Prices {
		marketFuture.Prices[k].High = -1
	}
	markets = append(markets, marketFuture)
	//言及のキャッシュと言及マップを作成
	mentionMap := map[*User][]*Price{}
	wg, mu := &sync.WaitGroup{}, &sync.Mutex{}
	for i, _ := range users {
		wg.Add(1)
		// TODO go
		go MentionUser(&users[i], markets, mentionMap, mu, wg)
	}
	wg.Wait()
	//発現数が多い人物を列挙する
	usersMap := map[*User]int{}

	for k, m := range mentionMap {
		//複数の株価言及を条件とする
		count := 0
		for _, p := range m {
			if p.High > 0 {
				count++
			}
		}
		if count > int(MentionRate*float32(len(markets)-1)) {
			if _, ok := usersMap[k]; ok == false {
				usersMap[k] = len(usersMap)
			}
		} else {
			delete(mentionMap, k)
		}
	}
	//言及マップの逆写像を作成
	mentionReverseMap := map[*Price][]*User{}
	for u, _ := range usersMap {
		for _, p := range mentionMap[u] {
			mentionReverseMap[p] = append(mentionReverseMap[p], u)
		}
	}
	//予測
	PredictRegression(usersMap, mentionReverseMap)
	PredictBayesian(mentionMap, mentionReverseMap, markets)
	var futureUsers []User
	var futurePrices []Price
	for u, _ := range usersMap {
		futureUsers = append(futureUsers, *u)
	}
	for p, _ := range mentionReverseMap {
		if p.High < 0 {
			futurePrices = append(futurePrices, *p)
		}
	}
	//選別
	sort.Slice(futureUsers, func(i, j int) bool { return math.Abs(futureUsers[i].Coefficient) > math.Abs(futureUsers[j].Coefficient) })
	sort.Slice(futurePrices, func(i, j int) bool { return futurePrices[i].PredictBayesian > futurePrices[j].PredictBayesian })
	for k, v := range futureUsers {
		fmt.Println(k, v.Name, v.Screen, v.Coefficient)
	}
	for k, v := range futurePrices {
		fmt.Println(k, v.Name, v.FullName, v.PredictRegression, v.PredictBayesian)
	}
	return futureUsers, futurePrices, math.Abs(futureUsers[0].Coefficient) < 1.0
}

/// @fn
/// ユーザーリストを更新し、予測する
func Prediction(users []User, markets []Market, prices []Price, future time.Time) (Predict, bool) {
	predict := Predict{
		Dead:   Deadline(future),
		Last:   future,
		Users:  users,
		Prices: prices,
	}
	//追加：10%増し
	predict.Users = AppendUsers(predict.Users, UsersLimit+UsersAdded)
	//学習と予測
	var isValid bool
	predict.Users, predict.Prices, isValid = Train(predict.Users, predict.Prices, future, markets)
	if len(predict.Users) > UsersLimit {
		predict.Users = predict.Users[:UsersLimit]
	}
	return predict, isValid
}

/// @fn 追加する
func AppendUsers(users []User, maxLength int) []User {
	rand.Seed(time.Now().UnixNano())
	result := map[int]*User{}
	for k, _ := range users {
		result[users[k].Id] = &users[k]
	}
	stock := map[int]*User{}
	count := 0
	for _, user := range result {
		if count += 1; count <= 2 {
			v := url.Values{}
			if user.Id != 0 {
				v.Set("user_id", strconv.Itoa(user.Id))
			} else {
				v.Set("screen_name", user.Screen)
			}
			v.Set("count", strconv.Itoa(100))
			v.Set("skip_status", "true")
			v.Set("include_user_entities", "false")
			if cursor, err := NewApi().GetFriendsList(v); err != nil {
				fmt.Printf("no friends", user.Name, user.Screen, user.Id)
			} else {
				for _, v := range cursor.Users {
					u := User{
						Id:          int(v.Id),
						Screen:      v.ScreenName,
						Name:        v.Name,
						Description: v.Description,
					}
					if v.Protected == false && IsValidUser(&u) {
						if _, ok := result[u.Id]; ok == false {
							stock[u.Id] = &u
						}
					}
				}
			}
		}
	}
	for _, v := range stock {
		if len(users) < maxLength{
			users = append(users, *v)
		}
	}
	fmt.Println("投資家候補の追加完了。", len(users))
	return users
}

func TestTwitter() {
	users := []User{
		{
			Id:     828661472,
			Screen: "lzpel",
			Name:   "lzpel",
		},
		{
			Id:     86075525,
			Screen: "_primenumber",
			Name:   "そすうぽよ",
		},
	}
	prices:=[]Price{
		{
			Code:     100,
			Name:     "真ん中あたり",
			FullName: "サピエンス",
			Open:     100,
			Close:    110,
		},
	}
	markets := []Market{
		{
			Born: time.Date(2021, time.June, 7, 12, 0, 0, 0, time.Local),
			Prices: prices,
		},
	}
	markets=markets
	r := new(regression.Regression)
	r.SetObserved("説明変数N+定数項1<=データ量")
	r.SetVar(0, "#1")
	r.SetVar(1, "#2")
	r.Train(regression.DataPoint(0.1, []float64{1, 0}))
	r.Train(regression.DataPoint(0.2, []float64{0, 1}))
	r.Train(regression.DataPoint(0.3, []float64{1, 1}))
	r.Run()
	fmt.Println(r.Formula)
	fmt.Println(225, false, HasReference("メタボリックシンドローム", &Price{Name: "ローム", FullName: "ローム"}))
	fmt.Println(425, true, HasReference("東京ドーム", &Price{Name: "東京ドーム", FullName: "東京ドーム"}))
	fmt.Println(300, true, HasReference("家畜ふん尿からＬＰガス家で使える燃料に古河電工", &Price{Name: "古河電", FullName: "古河電気工業"}))
	fmt.Println(200, false, HasReference("体が戦艦大和より硬いのにいきなり動くから", &Price{Name: "大和", FullName: "大和証券グループ本社"}))
	fmt.Println(225, false, HasReference("lại rồi ý, nay còn tụ tập siêu đông ntn", &Price{Name: "ＮＴＮ", FullName: "ＮＴＮ"}))
	fmt.Println(450, true, HasReference("おかげでH2Oリテイ、高島屋、Jフロント", &Price{Name: "Ｈ２Ｏリテイ", FullName: "エイチ・ツー・オーリテイリング"}))
	fmt.Println(time.Unix(PredictTweetTime(1401567948079190019, 0), 0).String(), "午前0:53 · 2021年6月7日")
	fmt.Println(MentionUpdate(&users[1],map[int64][]Price{Deadline(time.Now()).Unix():prices}))
	//Prediction(users, markets, markets[0].Prices, markets[0].Born)
}
