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
	//ツイート番号と株価番号の組
	MentionTime []time.Time
	Mention      []int64
	Coefficient  float64
}

type Predict struct {
	Self   *datastore.Key `datastore:"__key__"`
	Born   time.Time
	Users  []User `datastore:",noindex"`
	Prices []Price
}
