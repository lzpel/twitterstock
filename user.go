package main

import (
	"encoding/csv"
	"fmt"
	"github.com/ChimeraCoder/anaconda"
	"log"
	"math/rand"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func NewApi() *anaconda.TwitterApi {
	return anaconda.NewTwitterApiWithCredentials("828661472-lMGvALleFfL15jIkFhCgvhhph4ZpMKYlI573cI7a", "Kqch8IsgLBXSbhYvUVzDYSttA4LVl0pcUo55GSVA3CEMT", "mCPWWS6PazLgF36jkUnG4oPIT", "deqbZQubFuYWTWznQV2WfNL93sXoOSyZZsnkefJABb1taXwhYP")
}
func Min(x, y int) int {
	if x > y {
		return y
	}
	return x
}
func ShuffleUsers(returnUsers []User)[]User{
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(returnUsers), func(i, j int) {
		returnUsers[i], returnUsers[j] = returnUsers[j], returnUsers[i]
	})
	return returnUsers
}
func SaveUsers(csvpath string, users []User){
	file, e := os.OpenFile(csvpath, os.O_WRONLY|os.O_CREATE, 0666)
	if e != nil {
		log.Fatal(e)
	}
	defer file.Close()
	for _, user := range users {
		fmt.Fprintf(file,"%d,%s,%s\n", user.Id, user.Screen, user.Name)
	}
}
func LoadUsers(csvpath string) []User{
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
	returnUsers := make([]User,0,len(rows))
	for _, row := range rows {
		if len(row)==3{
			code,_ := strconv.Atoi(row[0])
			returnUsers = append(returnUsers, User{
				Name: row[2],
				Screen: row[1],
				Id: code,
			})
		}
	}
	return returnUsers
}
func FetchUsers(count int, seedusers []User, include, exclude []string) []User {
	api := NewApi()
	returnUsers := []User{}
	seedusers=ShuffleUsers(seedusers)
	for i := 0; i < Min(5, len(seedusers)); i++ {
		user:=seedusers[i]
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
						Name: n.Name,
						Id: int(n.Id),
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
func UpdateTexts(users []User){
	api:=NewApi()
	for _,user:=range users{
		v := url.Values{}
		if user.Id>0{
			v.Set("user_id", strconv.Itoa(user.Id))
		}else{
			v.Set("screen_name", user.Screen)
		}
		api.GetUserTimeline(v)
	}
}