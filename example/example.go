// radix example program.

package main

import (
	"fmt"
	"github.com/fzzbt/radix/redis"
	"strconv"
	"time"
)

func main() {
	var c *redis.Client
	var err error

	c, err = redis.NewClient(redis.Configuration{
		Database: 8,
		// Timeout in seconds
		Timeout: 10,

		// Custom TCP/IP address or Unix path.
		// Path: "/tmp/redis.sock",
		// Address: "127.0.0.1:6379",
	})

	if err != nil {
		fmt.Println("NewClient failed:", err)
	}

	defer c.Close()

	//** Blocking calls
	rep := c.Flushdb()
	if rep.Error != nil {
		fmt.Println("redis:", rep.Error)
		return
	}

	//* Strings

	// It's generally good idea to check for errors like this,
	// but for the sake of keeping this example short we'll omit these from now on.
    if rep = c.Set("mykey0", "myval0"); rep.Error != nil {
		fmt.Println("redis:", rep.Error)
		return
	}

	s, err := c.Get("mykey0").Str()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("mykey0:", s)

	myhash := map[string]string{
		"mykey1": "myval1",
		"mykey2": "myval2",
		"mykey3": "myval3",
	}

	// Alternatively:
	// c.Mset("mykey1", "myval1", "mykey2", "myval2", "mykey3", "myval3")
	c.Mset(myhash)

	ls, err := c.Mget("mykey1", "mykey2", "mykey3").List()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("mykeys values:", ls)

	//* List handling
	mylist := []string{"foo", "bar", "qux"}

	// Alternativaly:
	// c.Rpush("mylist", "foo", "bar", "qux")
	c.Rpush("mylist", mylist)

	mylist, err = c.Lrange("mylist", 0, -1).List()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("mylist:", mylist)

	//* Hash handling

	// Alternatively:
	// c.Hmset("myhash", ""mykey1", "myval1", "mykey2", "myval2", "mykey3", "myval3")
	c.Hmset("myhash", myhash)

	myhash, err = c.Hgetall("myhash").Hash()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("myhash:", myhash)

	//* Multicalls
	rep = c.MultiCall(func(mc *redis.MultiCall) {
		mc.Set("multikey", "multival")
		mc.Get("multikey")
	})

	// Multicall replies are guaranteed to have the same number of elements as in the call.
	// They can be accessed through Reply.Elems.
	s, err = rep.Elems[1].Str()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("multikey:", s)

	//* Transactions
	rep = c.Transaction(func(mc *redis.MultiCall) {
		mc.Set("trankey", "tranval")
		mc.Get("trankey")
	})

	s, err = rep.Elems[1].Str()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("trankey:", s)

	//* Complex transactions
	//  Atomic INCR replacement with transactions
	myIncr := func(key string) *redis.Reply {
		return c.MultiCall(func(mc *redis.MultiCall) {
			var curval int

			mc.Watch(key)
			mc.Get(key)
			rep := mc.Flush()
			s, err := rep.Elems[1].Str()
			if err == nil {
				curval, err = strconv.Atoi(s)
			}
			nextval := curval + 1

			mc.Multi()
			mc.Set(key, nextval)
			mc.Exec()
		})
	}

	myIncr("ctrankey")
	myIncr("ctrankey")
	myIncr("ctrankey")

	s, err = c.Get("ctrankey").Str()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("ctrankey:", s)

	//** Asynchronous calls
	c.Set("asynckey", "asyncval")
	fut := c.AsyncGet("asynckey")

	// do something here

	// block until reply is available
	s, err = fut.Reply().Str()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("asynckey:", s)

	//* Pub/sub
	msgHdlr := func(msg *redis.Message) {
		switch msg.Type {
		case redis.MessageMessage:
			fmt.Printf("Received message \"%s\" from channel \"%s\".\n", msg.Payload, msg.Channel)
		case redis.MessagePmessage:
			fmt.Printf("Received pattern message \"%s\" from channel \"%s\" with pattern "+
				"\"%s\".\n", msg.Payload, msg.Channel, msg.Pattern)
		default:
			fmt.Println("Received other message:", msg)
		}
	}

	sub, errr := c.Subscription(msgHdlr)
	if errr != nil {
		fmt.Printf("Failed to subscribe: '%s'!\n", errr)
		return
	}

	defer sub.Close()

	sub.Subscribe("chan1", "chan2")
	sub.Psubscribe("chan*")

	c.Publish("chan1", "foo")
	sub.Unsubscribe("chan1")
	c.Publish("chan2", "bar")

	// give some time for the message handler to receive the messages
	time.Sleep(time.Second)
}
