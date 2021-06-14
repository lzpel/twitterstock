package main

import (
	"cloud.google.com/go/datastore"
	"time"
)

// Do not change or commit

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
	Prices []Price
}

type User struct {
	Id           int
	Screen, Name string
	Mention      []int64
	//（キャッシュ時刻のUnix秒（存在確認と寿命管理にのみ使う）:6byte、株価番号:2byte）、ツイートID：8byteの組。番兵含む
	Coefficient  float64
}

type Predict struct {
	Born   time.Time
	Users  []User `datastore:",noindex"`
	Prices []Price
}

func (p*Predict) Key() *datastore.Key{
	return NewIdKey("PREDICT",p.Born.Unix())
}
