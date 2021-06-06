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
	const PREDICT_HOUR = 7
	t = t.In(time.Local).Add(-time.Hour * time.Duration(PREDICT_HOUR))
	return time.Date(t.Year(), t.Month(), t.Day(), PREDICT_HOUR, 0, 0, 0, time.Local)
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
	if strings.Contains(tw, ns) || strings.Contains(tw, nl) {
		if IsValidName(ns) && IsValidName(nl) {
			return true
		}
	}
	return false
}

/// @fn
/// time.Localで指定された時間帯でtimeを含む一日に含まれる言及ツイートと言及株式番号
func UserMention(user *User, dict []Price, day time.Time, useCache bool) {
	day = Daily(day)
	if useCache{
		for _,v:=range user.MentionTime{
			if v==day{
				return
			}
		}
	}
	MentionTime:=user.MentionTime
	Mention:=user.Mention
	if MentionTime == nil {
		MentionTime = make([]time.Time, 0)
	}
	if Mention == nil {
		Mention = make([]int64, 0, 32)
	}
	MentionTime=append(MentionTime,day)
	v := url.Values{}
	v.Set("user_id", strconv.Itoa(user.Id))
	v.Set("max_id", strconv.FormatInt(PredictTweetTime(0, day.Unix()), 10))
	v.Set("since_id", strconv.FormatInt(PredictTweetTime(0, day.Add(-24 * time.Hour).Unix()), 10))
	v.Set("count", strconv.Itoa(200))
	v.Set("exclude_replies", "false")
	if tweets, err := NewApi().GetUserTimeline(v); err != nil {
		fmt.Printf("no tweets https://twitter.com/intent/user?user_id=%v\n", user.Id)
	} else {
		for _, tweet := range tweets {
			PredictTweetTimeUpdate(&tweet)
			for _, p := range dict {
				if HasReference(tweet.FullText, &p) {
					Mention = append(Mention, tweet.Id, int64(p.Code))
				}
			}
		}
	}
	//保存
	user.MentionTime = make([]time.Time, 0, len(MentionTime))
	user.Mention = make([]int64, 0, len(Mention))
	MinTime:=time.Now().Add(- time.Hour * 24 * 5)
	MinId := PredictTweetTime(0, MinTime.Unix())
	for _,v :=range MentionTime{
		if v.Unix() > MinTime.Unix(){
			user.MentionTime=append(user.MentionTime, v)
		}
	}
	for k := 0; k < len(user.Mention); k += 2 {
		if user.Mention[k] > MinId {
			user.Mention = append(user.Mention, user.Mention[k], user.Mention[k+1])
		}
	}
}

func IsValidPrice(v *Price) bool {
	if v.Close > 10 && v.Open > 10 {
		//浮動小数点読み込みミスを暫定的に除外
		x := float64(float32(v.Close-v.Open)/float32(v.Open) - 1)
		if 0.5 > math.Abs(x) {
			v.Value = x
		}
	}
	return false;
}

func Vars(users *[]User, prices *[]Price, t time.Time) map[int][]float64 {
	data := map[int][]float64{}
	MinID, MaxID := PredictTweetTime(0, t.Unix()), PredictTweetTime(0, t.Add(time.Hour * 24).Unix())
	for k, u := range *users {
		for i := 0; i < len(u.Mention); i++ {
			id, code := u.Mention[i], int(u.Mention[i+1])
			if MinID < id && id < MaxID {
				vars, ok := data[code]
				if ok == false {
					vars = make([]float64, len(*users))
				}
				vars[k] = 1
				data[code] = vars
			}
		}
	}
	return data
}

/// @fn
func Train(users *[]User, markets *[]Market) {
	r := new(regression.Regression)
	r.SetObserved("株価の変動率")
	for k, u := range *users {
		r.SetVar(k, u.Name)
	}
	for k, m := range *markets {
		vars := Vars(users, &m.Prices, m.Born)
		prices := map[int]*Price{}
		if len(*markets)-k > 1 {
			for _, p := range m.Prices {
				if IsValidPrice(&p) {
					prices[p.Code] = &p
				}
			}
			for k, d := range vars {
				if p, ok := prices[k]; ok == true {
					r.Train(regression.DataPoint(p.Value, d))
				}
			}
		} else {
			r.Run()
			for k, v := range vars {
				prices[k].Value, _ = r.Predict(v)
			}
			for k, _ := range *users {
				(*users)[k].Coefficient = r.Coeff(k + 1)
			}
		}
	}
}

/// @fn
/// ユーザーリストを更新し、予測する
func Prediction(ul []User, markets []Market, useCache bool) Predict {
	//最後は予測
	predict := Market{
		Born: Daily(time.Now()),
	}
	copy(predict.Prices, markets[0].Prices)
	markets = append(markets, predict)
	//追加：2割増し
	ul = AppendUsers(ul, UsersLimit/5)
	//非同期キャッシュ
	wg := &sync.WaitGroup{}
	for i, _ := range ul {
		wg.Add(1)
		func(u *User, ms []Market) {
			defer wg.Done()
			for _, m := range ms {
				UserMention(u, m.Prices, m.Born,useCache)
			}
		}(&ul[i], markets)
	}
	wg.Wait()
	//学習と予測
	Train(&ul, &markets)
	//厳選
	sort.Slice(ul, func(i, j int) bool { return ul[i].Coefficient > ul[j].Coefficient })
	sort.Slice(predict.Prices, func(i, j int) bool { return predict.Prices[i].Value > predict.Prices[j].Value })
	if len(ul) > UsersLimit {
		ul = ul[:UsersLimit]
	}
	//表示
	for _,v:=range ul{
		fmt.Println(v.Screen,v.Name,v.Coefficient)
	}
	for _,v:=range predict.Prices {
		fmt.Println(v.Name, v.Value)
	}
	return Predict{
		Born:   predict.Born,
		Users:  ul,
		Prices: predict.Prices,
	}
}

/// @fn 追加する
func AppendUsers(users []User, count int) []User {
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
	return users
}

func TestTwitter() {
	users := []User{
		{
			Id:     1284381210429415425,
			Screen: "Sapiensism",
		},
	}
	markets := []Market{
		{
			Born: time.Now(),
			Prices: []Price{
				{
					Code:     100,
					Name:     "霊長類",
					FullName: "サピエンス",
					Diff:     100,
				},
				{
					Code:     101,
					Name:     "進化",
					FullName: "遺伝",
					Diff:     100,
				},
				{
					Code:     102,
					Name:     "人間",
					FullName: "ヒト",
				},
			},
		},
	}
	fmt.Println(false, HasReference("メタボリックシンドローム", &Price{Name: "ローム", FullName: "ローム"}))
	fmt.Println(true, HasReference("東京ドーム", &Price{Name: "東京ドーム", FullName: "東京ドーム"}))
	fmt.Println(true, HasReference("家畜ふん尿からＬＰガス家で使える燃料に古河電工", &Price{Name: "古河電", FullName: "古河電気工業"}))
	fmt.Println(false, HasReference("体が戦艦大和より硬いのにいきなり動くから", &Price{Name: "大和", FullName: "大和証券グループ本社"}))
	fmt.Println(false, HasReference("lại rồi ý, nay còn tụ tập siêu đông ntn", &Price{Name: "ＮＴＮ", FullName: "ＮＴＮ"}))
	fmt.Println(true, HasReference("おかげでH2Oリテイ、高島屋、Jフロント", &Price{Name: "Ｈ２Ｏリテイ", FullName: "エイチ・ツー・オーリテイリング"}))
	UserMention(&users[0], markets[0].Prices, time.Now(),false)
	fmt.Println(users[0].Mention)
	Prediction(users, markets, false)
}
