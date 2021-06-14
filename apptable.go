package main

import (
	"cloud.google.com/go/datastore"
	"context"
	"fmt"
	"os"
	"time"
)

func NewClient() (*datastore.Client, context.Context) {
	ctx := context.Background()
	c, e := datastore.NewClient(ctx, os.Getenv("PROJECT_ID"))
	if e != nil {
		panic(e)
	}
	return c, ctx
}
func NewQuery(kind string) *datastore.Query {
	return datastore.NewQuery(kind)
}
func NewKey(kind string) *datastore.Key {
	return datastore.IncompleteKey(kind, nil)
}
func NewIdKey(kind string,id int64) *datastore.Key {
	return datastore.IDKey(kind,id,nil)
}
func TablePut(k *datastore.Key, v interface{}) *datastore.Key {
	if c, x := NewClient(); c != nil {
		defer c.Close()
		if k, err := c.Put(x, k, v); err != nil {
			panic(err)
		} else {
			return k
		}
	}
	return nil
}
func TableGet(k *datastore.Key, v interface{}) *datastore.Key {
	if c, x := NewClient(); c != nil {
		defer c.Close()
		if err := c.Get(x, k, v); err != nil {
			panic(err)
		} else {
			return k
		}
	}
	return nil
}
func TableGetAll(q *datastore.Query, v interface{}) []*datastore.Key {
	if c, x := NewClient(); c != nil {
		defer c.Close()
		if keys, err := c.GetAll(x, q, v); err != nil {
			panic(err)
		} else {
			return keys
		}
	}
	return nil
}
func TableDeleteAll(keys []*datastore.Key) error{
	if c, x := NewClient(); c != nil {
		defer c.Close()
		if err := c.DeleteMulti(x,keys); err != nil {
			return err
		}
	}
	return nil
}
func TableCount(q *datastore.Query) int {
	if c, x := NewClient(); c != nil {
		defer c.Close()
		if n, err := c.Count(x, q); err != nil {
			panic(err)
		} else {
			return n
		}
	}
	return -1
}
func TableDemo() {
	type ExampleEntity struct {
		Self *datastore.Key `datastore:"__key__"`
		Born time.Time
		Age  int
	}
	e := ExampleEntity{
		Born: time.Now(),
		Age:  24,
	}
	k := TablePut(NewKey("EXAMPLE"), &e)
	m := ExampleEntity{}
	TableGet(k, &m)
	ms := []ExampleEntity{}
	TableGetAll(NewQuery("EXAMPLE").Order("-Born").Limit(3), &ms)
	fmt.Println(m)
}
