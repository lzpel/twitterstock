package main

import (
	"fmt"
	"github.com/ChimeraCoder/anaconda"
	"github.com/sajari/regression"
	"golang.org/x/text/unicode/norm"
	"math"
	"math/rand"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

/// @file
/// 日付から確率分布を算出する
/// 相場から正解の確立分布を算出する
/// 相関を採りランキングを更新する
/// 学術的に意味がある簡単なデータ構造を引数や戻り値に設定すると使いまわしやすい

const (
	SeedUserId = 1086060182860292096 //ぱちょ@h2bl0cker_
	UsersLimit = 50
	//株価と関係ないツイートの例
	//楽天証券 サイバーマンデー ホワイトハウス 平和賞 #ローソンから一足早くお届け #武田愛奈 ライオン アンソニー・ファウチ所長
)

// @fn
// API取得
func NewApi() *anaconda.TwitterApi {
	return anaconda.NewTwitterApiWithCredentials("828661472-lMGvALleFfL15jIkFhCgvhhph4ZpMKYlI573cI7a", "Kqch8IsgLBXSbhYvUVzDYSttA4LVl0pcUo55GSVA3CEMT", "mCPWWS6PazLgF36jkUnG4oPIT", "deqbZQubFuYWTWznQV2WfNL93sXoOSyZZsnkefJABb1taXwhYP")
}

/// @variable
var PredictTweet = []int64{0, 0, 0, 0}

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
		tweets, _ := NewApi().GetUserTimeline(v)
		for _, v := range tweets {
			PredictTweetTimeUpdate(&v)
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

func Daily(t time.Time) time.Time {
	// 東証の取引時間は現在、午前９時―午後３時で、途中１時間の休憩が入る。
	const PredictHour = 8
	t = t.In(time.Local).Add(-time.Hour * time.Duration(PredictHour))
	return time.Date(t.Year(), t.Month(), t.Day(), PredictHour, 0, 0, 0, time.Local)
}

func FormatString(s string) string {
	return norm.NFKC.String(s)
}

func IsValidName(s string) bool {
	const IgnoreWords = "トレンドサイバーローソン"
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
	if strings.Contains(IgnoreWords, s) {
		score = 0
	}
	return score >= 300
}

func HasReference(s string, p *Price) bool {
	tw, ns, nl := FormatString(s), FormatString(p.Name), FormatString(p.FullName)
	if (strings.Contains(tw, ns) && IsValidName(ns)) || (strings.Contains(tw, nl) && IsValidName(nl)) {
		return true
	}
	return false
}

func IsValidPrice(v *Price) bool {
	//浮動小数点読み込みミスを暫定的に除外
	if v.Close > 10 && v.Open > 10 {
		x := float64(float32(v.Close)/float32(v.Open) - 1)
		if 0.5 > math.Abs(x) {
			v.Value = x
			return true
		}
	}
	return false
}

func UserMentionVector(users []User, k int, dict []Price, dayUnix, minID, maxID, maxAge int64, output map[int][]float64) {
	u, cacheSkip := &users[k], false
	for _, v := range u.Mention {
		if (v >> 16) == dayUnix {
			fmt.Println("Cached")
			cacheSkip = true
			break
		}
	}
	if cacheSkip == false {
		u.Mention = append(u.Mention, dayUnix<<16+int64(0), 0)
		v := url.Values{}
		v.Set("user_id", strconv.Itoa(u.Id))
		v.Set("max_id", strconv.FormatInt(maxID, 10))
		v.Set("since_id", strconv.FormatInt(minID, 10))
		v.Set("count", strconv.Itoa(200))
		v.Set("exclude_replies", "false")
		if tweets, err := NewApi().GetUserTimeline(v); err != nil {
			fmt.Printf("no tweets https://twitter.com/intent/user?user_id=%v\n", u.Id)
		} else {
			for _, tweet := range tweets {
				PredictTweetTimeUpdate(&tweet)
				for _, p := range dict {
					if HasReference(tweet.FullText, &p) {
						u.Mention = append(u.Mention, dayUnix<<16+int64(p.Code), tweet.Id)
					}
				}
			}
		}
		fmt.Println(u.Name, "Done")
	} else {
		fmt.Println(u.Name, "Skipped")
	}
	Mention := make([]int64, 0, len(u.Mention))
	for i := 0; i < len(u.Mention); i += 2 {
		unix, code, id := u.Mention[i]>>16, int(u.Mention[i]&0xffff), u.Mention[i+1]
		if minID < id && id < maxID && code != 0 {
			vars, ok := output[code]
			if ok == false {
				vars = make([]float64, len(users))
			}
			vars[k] = 1.0
			output[code] = vars
		}
		if unix > maxAge {
			Mention = append(Mention, u.Mention[i], u.Mention[i+1])
		}
	}
	u.Mention = Mention
}

func UserMentionVectors(users []User, dict []Price, day time.Time) (map[int][]float64, map[int]*Price) {
	// X行列
	day = Daily(day)
	MinID, MaxID := PredictTweetTime(0, day.Add(-time.Hour * 24).Unix()), PredictTweetTime(0, day.Unix())
	MaxAge := time.Now().Add(- time.Hour * 24 * 5)
	output := map[int][]float64{}
	wg := &sync.WaitGroup{}
	for k, _ := range users {
		wg.Add(1)
		//TODO: go func
		func() {
			defer wg.Done()
			UserMentionVector(users, k, dict, day.Unix(), MinID, MaxID, MaxAge.Unix(), output)
		}()
	}
	wg.Wait()
	// Yベクトル
	prices := map[int]*Price{}
	for k, _ := range dict {
		if IsValidPrice(&dict[k]) {
			prices[dict[k].Code] = &dict[k]
		}
	}
	return output, prices
}

/// @fn
func Train(users []User, markets []Market, predict []Price, future time.Time) {
	r := new(regression.Regression)
	r.SetObserved("株価の変動率")
	for k, u := range users {
		r.SetVar(k, u.Name)
	}
	for _, m := range markets {
		vars, prices := UserMentionVectors(users, m.Prices, m.Born)
		for k, d := range vars {
			if p, ok := prices[k]; ok == true {
				r.Train(regression.DataPoint(p.Value, d))
			}
		}
	}
	vars, prices := UserMentionVectors(users, predict, future)
	r.Run()
	for k, v := range vars {
		var e error
		if prices[k].Value, e = r.Predict(v); e != nil {
			prices[k].Value = 0
		}
	}
	for k, _ := range users {
		users[k].Coefficient = r.Coeff(k + 1)
	}
}

/// @fn
/// ユーザーリストを更新し、予測する
func Prediction(users []User, markets []Market, prices []Price, future time.Time, useCache bool) Predict {
	//最後は予測
	// TODO: Deep Copy
	predict := Predict{
		Born:   Daily(future),
		Users:  users,
		Prices: prices,
	}
	//追加：2割増し
	AppendUsers(predict.Users, UsersLimit/5)
	//学習と予測
	Train(predict.Users, markets, predict.Prices, predict.Born)
	//厳選
	sort.Slice(predict.Users, func(i, j int) bool { return predict.Users[i].Coefficient > predict.Users[j].Coefficient })
	sort.Slice(predict.Prices, func(i, j int) bool { return predict.Prices[i].Value > predict.Prices[j].Value })
	if len(predict.Users) > UsersLimit {
		predict.Users = predict.Users[:UsersLimit]
	}
	//表示
	for _, v := range predict.Users {
		fmt.Println(v.Screen, v.Name, v.Coefficient)
	}
	for _, v := range predict.Prices {
		fmt.Println(v.Name, v.Value)
	}
	return predict
}

/// @fn 追加する
func AppendUsers(users []User, count int) {
	rand.Seed(time.Now().UnixNano())
	//全ユーザーから一人を選出
	idx := rand.Intn(len(users))
	id := users[idx].Id
	// フォローを抽出
	v := url.Values{}
	v.Set("user_id", strconv.Itoa(id))
	v.Set("count", "1000")
	if cursor, err := NewApi().GetFriendsIds(v); err != nil {
		fmt.Printf("no friends https://twitter.com/intent/user?user_id=%v\n", id)
	} else {
		ids := cursor.Ids
		for i := 0; i < count; i++ {
			id := ids[rand.Intn(len(ids))]
			for _, u := range users {
				if int64(u.Id) == id {
					id = 0
				}
			}
			if id != 0 {
				users = append(users, User{
					Id: int(id),
				})
			}
		}
	}
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
	markets := []Market{
		{
			Born: time.Date(2021, time.June, 7, 12, 0, 0, 0, time.Local),
			Prices: []Price{
				{
					Code:     100,
					Name:     "誕生日",
					FullName: "サピエンス",
					Open:     100,
					Close:    110,
				},
				{
					Code:     101,
					Name:     "ABC全",
					FullName: "霊長類",
					Open:     100,
					Close:    120,
				},
				{
					Code:     102,
					Name:     "部分和問題",
					FullName: "行列累乗",
					Open:     100,
					Close:    130,
				},
			},
		},
	}
	r := new(regression.Regression)
	r.SetObserved("説明変数N+定数項1<=データ量")
	r.SetVar(0, "#1")
	r.SetVar(1, "#2")
	r.Train(regression.DataPoint(1.0, []float64{1.0, 0, 0}))
	r.Train(regression.DataPoint(2.0, []float64{0, 1.0, 0}))
	r.Train(regression.DataPoint(4.0, []float64{1.0, 0, 1.0}))
	r.Train(regression.DataPoint(0.0, []float64{0.0, 0, 0.0}))
	r.Run()
	fmt.Println(r.Formula)
	fmt.Println(r)
	return
	fmt.Println(225, false, HasReference("メタボリックシンドローム", &Price{Name: "ローム", FullName: "ローム"}))
	fmt.Println(425, true, HasReference("東京ドーム", &Price{Name: "東京ドーム", FullName: "東京ドーム"}))
	fmt.Println(300, true, HasReference("家畜ふん尿からＬＰガス家で使える燃料に古河電工", &Price{Name: "古河電", FullName: "古河電気工業"}))
	fmt.Println(200, false, HasReference("体が戦艦大和より硬いのにいきなり動くから", &Price{Name: "大和", FullName: "大和証券グループ本社"}))
	fmt.Println(225, false, HasReference("lại rồi ý, nay còn tụ tập siêu đông ntn", &Price{Name: "ＮＴＮ", FullName: "ＮＴＮ"}))
	fmt.Println(450, true, HasReference("おかげでH2Oリテイ、高島屋、Jフロント", &Price{Name: "Ｈ２Ｏリテイ", FullName: "エイチ・ツー・オーリテイリング"}))
	fmt.Println(time.Unix(PredictTweetTime(1401567948079190019, 0), 0).String(), "午前0:53 · 2021年6月7日")
	x, y := UserMentionVectors(users, markets[0].Prices, markets[0].Born)
	for k, m := range x {
		fmt.Println(k, y[k].Value, m)
	}
	Prediction(users, markets, markets[0].Prices, markets[0].Born, true)
}
