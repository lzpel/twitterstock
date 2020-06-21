package main

import (
	"github.com/ChimeraCoder/anaconda"
	"log"
	"math/rand"
	"net/url"
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
func GetMatchedUsers(count int, users []User, include, exclude []string) []User {
	api := NewApi()
	returnUsers := []User{}
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < Min(5, len(users)); i++ {
		user:=users[rand.Int()%len(users)]
		v := url.Values{}
		if user.Id>0{
			v.Set("user_id", strconv.Itoa(user.Id))
		}else{
			v.Set("screen_name", user.Screen)
		}
		v.Set("count", strconv.Itoa(50))
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
						Id: int(n.Id),
					})
				}
			}
		}
	}
	rand.Shuffle(len(returnUsers), func(i, j int) {
		returnUsers[i], returnUsers[j] = returnUsers[j], returnUsers[i]
	})
	if count > 0 {
		returnUsers = returnUsers[:Min(len(returnUsers), count)]
	}
	for i, n:=range returnUsers{
		log.Println(i,n)
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