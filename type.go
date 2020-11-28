package main

import (
	"cloud.google.com/go/datastore"
	"encoding/json"
	"time"
)

// Do not change or commit

/// 各銘柄の上昇確率分布
type Possibility = map[int]float32
type MapPossibility map[int]Possibility
type Json = []byte

type Price struct {
	Name, FullName               string
	Code                         int
	Market                       rune //1,2,j,m
	Open, Close, High, Low, Diff int
}

type Market struct {
	Self   *datastore.Key `datastore:"__key__"`
	Born   time.Time
	Prices []Price
}

type User struct {
	Id           int
	Screen, Name string
	Cache        Json
}

type Predict struct {
	Self             *datastore.Key `datastore:"__key__"`
	Born             time.Time
	Users            []User `datastore:",noindex"`
	UsersPossibility Json   `datastore:",noindex"`
	Codes            []int  `datastore:",noindex"`
	CodesPossibility Json   `datastore:",noindex"`
}

func ToData(p interface{}, j Json) interface{} {
	if err := json.Unmarshal(j, p); err != nil {
		return nil
	} else {
		return p
	}
}
func ToJson(p interface{}) Json {
	if bytes, err := json.Marshal(p); err != nil {
		return nil
	} else {
		return bytes
	}
}
