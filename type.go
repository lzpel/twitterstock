package main

import "time"
// no change
// no commit
type Price struct {
	Name                         string
	Code                         int
	Market                       rune //1,2,j,m
	Open, Close, High, Low, Diff int
}
type Market struct {
	Born   time.Time
	Prices []Price
}

