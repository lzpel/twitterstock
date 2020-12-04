package main

import (
	"fmt"
	"github.com/ChimeraCoder/anaconda"
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
	SeedUserId  = 1086060182860292096 //ぱちょ@h2bl0cker_
	UsersLimit  = 50
	//株価と関係ないツイートの例
	//楽天証券 サイバーマンデー ホワイトハウス 平和賞 #ローソンから一足早くお届け #武田愛奈 ライオン アンソニー・ファウチ所長
)

// @fn
// API取得
func NewApi() *anaconda.TwitterApi {
	return anaconda.NewTwitterApiWithCredentials("828661472-lMGvALleFfL15jIkFhCgvhhph4ZpMKYlI573cI7a", "Kqch8IsgLBXSbhYvUVzDYSttA4LVl0pcUo55GSVA3CEMT", "mCPWWS6PazLgF36jkUnG4oPIT", "deqbZQubFuYWTWznQV2WfNL93sXoOSyZZsnkefJABb1taXwhYP")
}

/// @variable
var PredictTweet, PredictTweetFrom anaconda.Tweet

// @fn
// unixtimeに最も近い時刻のツイートのIDを予測する関数
// 一次近似を仮定しているが十分正確
func PredictId(tweet *anaconda.Tweet, unix int64) int64 {
	if tweet != nil {
		if create, err := tweet.CreatedAtTime(); err != nil {
			panic(err)
		} else {
			tweet.QuotedStatusID = create.Unix()
		}
		if PredictTweet.Id == 0 || PredictTweet.Id < tweet.Id {
			PredictTweet = *tweet
		}
		if (PredictTweet.QuotedStatusID - tweet.QuotedStatusID) > 3600*6 {
			if PredictTweetFrom.Id == 0 || PredictTweetFrom.Id < tweet.Id {
				PredictTweetFrom = *tweet
			}
		}
	}
	if unix != 0 {
		if PredictTweet.Id == 0 || PredictTweetFrom.Id == 0 || PredictTweetFrom.Id == PredictTweet.Id {
			v := url.Values{}
			v.Set("user_id", "783214")
			v.Set("count", strconv.Itoa(200))
			v.Set("trim_user", "true")
			v.Set("exclude_replies", "false")
			tweets, _ := NewApi().GetUserTimeline(v)
			for _, v := range tweets {
				PredictId(&v, 0)
			}
		}
		return PredictTweetFrom.Id + int64(float64(unix-PredictTweetFrom.QuotedStatusID)*float64(PredictTweet.Id-PredictTweetFrom.Id)/float64(PredictTweet.QuotedStatusID-PredictTweetFrom.QuotedStatusID))
	}
	return 0
}

func Daily(t time.Time) time.Time {
	const PREDICT_HOUR = 8
	t = t.In(time.Local).Add(-time.Hour * time.Duration(PREDICT_HOUR))
	return time.Date(t.Year(), t.Month(), t.Day(), PREDICT_HOUR, 0, 0, 0, time.Local)
}

func HasReference(s string,p*Price) bool{
	FormatString:=func(s string) string{
		//空文字列
		if len(s)==0{
			return "空文字列含有判定回避"
		}
		//全角英数字
		return strings.ToUpper(string(norm.NFKC.Bytes([]byte(s))))
	}
	IsValid:=func(s string) bool{
		const IgnoreWords = "楽天トレンドサイバーハウス平和ローソングリー武田イオンソニー"
		score:=0
		for _,c:=range s{
			if (c >= 'A' && c <= 'Z')||(c >= '0' && c <= '9'){
				score+=75// 300/4==75
			}
			if unicode.In(c, unicode.Hiragana)||unicode.In(c, unicode.Katakana){
				score+=75
			}
			score+=100
		}
		if strings.Contains(IgnoreWords,s){
			score=0
		}
		return score>=300
	}
	tw,ns,nl:=FormatString(s),FormatString(p.Name),FormatString(p.FullName)
	if strings.Contains(tw, ns) || strings.Contains(tw, nl) {
		if IsValid(ns) && IsValid(nl){
			return true
		}
	}
	return false
}

/// @fn
/// time.Localで指定された時間帯でtimeを含む一日につぶやかれたUserのツイートから確率分布を抽出する
func UserPossibility(user *User, dict []Price, day time.Time, useCache bool) Possibility {
	//時刻
	day=Daily(day)
	//蓄積
	cache := MapPossibility{}
	ToData(&cache, user.Cache)
	if useCache{
		if v, ok := cache[int(day.Unix())]; ok && v != nil {
			return v
		}
	}
	r := Possibility{}
	//収集
	v := url.Values{}
	v.Set("user_id", strconv.Itoa(user.Id))
	v.Set("max_id", strconv.FormatInt(PredictId(nil, day.Unix()), 10))
	v.Set("since_id", strconv.FormatInt(PredictId(nil, day.Add(-24 * time.Hour).Unix()), 10))
	v.Set("count", strconv.Itoa(200))
	v.Set("exclude_replies", "false")
	if tweets, err := NewApi().GetUserTimeline(v); err != nil {
		fmt.Printf("no tweets https://twitter.com/intent/user?user_id=%v\n", user.Id)
	} else {
		for _, tweet := range tweets {
			PredictId(&tweet, 0)
			for _, p := range dict {
				if HasReference(tweet.FullText, p) {
					r[p.Code] += 1
				}
			}
		}
		//正規化
		Normalize(r, true)
	}
	//保存
	for k, _ := range cache {
		if k < int(time.Now().Add(- time.Hour * 24 * 7).Unix()) {
			delete(cache, k)
		}
	}
	cache[int(day.Unix())] = r
	user.Cache = ToJson(cache)
	return r
}

/// @fn
/// 相場から確率分布を得る
func MarketPossibility(prices []Price) Possibility {
	r := Possibility{}
	for _, v := range prices {
		if v.Close < 10 || v.Open < 10 {
			continue //浮動小数点読み込みミスを暫定的に除外
		}
		if x := float32(v.Close-v.Open) / float32(v.Open); 0.5 > math.Abs(float64(x)) {
			r[v.Code] = x
		} else {
			fmt.Println("Outliers", v)
		}
	}
	Normalize(r, false)
	return r
}

/// @fn
func Integrate(m []Possibility) Possibility {
	r := Possibility{}
	for _, v := range m {
		for k, v := range v {
			r[k] += v
		}
	}
	return r
}

/// @fn
func Normalize(r Possibility, isDistribution bool) {
	sum := float32(0)
	for _, v := range r {
		sum += v
	}
	if isDistribution {
		for k, _ := range r {
			r[k] /= sum
		}
	} else {
		avg, std := sum/float32(len(r)), float32(0)
		for _, v := range r {
			std += (v - avg) * (v - avg)
		}
		std = float32(math.Sqrt(float64(std) / float64(len(r))))
		for k, v := range r {
			r[k] = (v - avg) / std
		}
	}
}

/// @fn
/// 確立分布の相関を採る
func Correlation(a, b Possibility) float32 {
	c := float32(0.0)
	for k, v := range a {
		c += b[k] * v
	}
	return c
}

/// @fn
/// ユーザーリストを更新し、予測する
func Prediction(ul []User, markets []Market,useCache bool) Predict {
	now := time.Now()
	//追加：上位2割
	ul = AppendUsers(ul, UsersLimit/5)
	//非同期キャッシュ
	wg := &sync.WaitGroup{}
	for i, _ := range ul {
		wg.Add(1)
		go func(u *User, ms []Market, t time.Time) {
			defer wg.Done()
			UserPossibility(u, ms[0].Prices, t,useCache)
			for _, m := range ms {
				UserPossibility(u, m.Prices, m.Born, useCache)
			}
		}(&ul[i], markets, now)
	}
	wg.Wait()
	//検証
	up := Possibility{}
	for _, m := range markets {
		mpp := MarketPossibility(m.Prices)
		for i, _ := range ul {
			//vではなくul[i]を使い複製を回避し参照を渡す
			mup := UserPossibility(&ul[i], m.Prices, m.Born, true)
			up[ul[i].Id] += Correlation(mpp, mup)
		}
	}
	Normalize(up, true)
	//厳選
	sort.Slice(ul, func(i, j int) bool { return up[ul[i].Id] > up[ul[j].Id] })
	if len(ul) > UsersLimit {
		ul = ul[:UsersLimit]
	}
	for _, v := range ul {
		fmt.Println(v.Id, up[v.Id])
	}
	//予測
	ppl := []Possibility{}
	for i := 0; i < len(ul); i++ {
		ppl = append(ppl, UserPossibility(&ul[i], markets[0].Prices, now,true))
	}
	cp := Integrate(ppl)
	Normalize(cp, true)
	cl := make([]int, 0, len(cp))
	for k, _ := range cp {
		cl = append(cl, int(k))
	}
	sort.Slice(cl, func(i, j int) bool { return cp[cl[i]] > cp[cl[j]] })
	for _, v := range cl {
		fmt.Println(v, cp[v])
	}
	return Predict{
		Born:             Daily(now),
		Users:            ul,
		UsersPossibility: ToJson(up),
		Codes:            cl,
		CodesPossibility: ToJson(cp),
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
	fmt.Println(Integrate([]Possibility{Possibility{1: 1, 2: 1}, {2: 1, 3: 2}}))
	fmt.Println(Correlation(Possibility{1: 0.5, 2: 0.5}, Possibility{2: 0.5, 3: 0.5}))
	fmt.Println(UserPossibility(&users[0], markets[0].Prices, time.Now(), false))
	fmt.Println(MarketPossibility(markets[0].Prices))
	Prediction(users, markets,false)
	fmt.Println()
}
