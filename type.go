package main

import (
	"cloud.google.com/go/datastore"
	"time"
)

const (
	SeedUserId        = 1086060182860292096 //ぱちょ@h2bl0cker_
	UsersLimit        = 100
	MarketDays        = 14
	CacheAge          = time.Hour * 24 * 30
	MentionRate    = 0.5
	ExcludePriceWords = "トレンドサイバーローソン"
	IncludeUserWords  = "株,運用,資産,投資,先物,銘柄,取引,相場,不動産"
)

type Price struct {
	Name, FullName               string
	Code                         int
	Market                       rune //1,2,j,m
	Open, Close, High, Low, Diff int
	Value                        float64
}

type Market struct {
	Self   *datastore.Key `datastore:"__key__"`
	Born   time.Time
	Prices []Price `datastore:",noindex"`
}

type User struct {
	Id           int
	Screen, Name string
	Description  string
	Mention      []int64
	//（キャッシュ時刻のUnix秒（存在確認と寿命管理にのみ使う）:6byte、株価番号:2byte）、ツイートID：8byteの組。番兵含む
	Coefficient float64
}

type Predict struct {
	Born, Last time.Time
	Users      []User  `datastore:",noindex"`
	Prices     []Price `datastore:",noindex"`
}

func (p *Predict) Key() *datastore.Key {
	return NewIdKey("PREDICT", p.Born.Unix())
}
