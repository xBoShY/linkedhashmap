/*
 * Copyright (c) 2023, Hugo Meneses
 * All rights reserved.
 *
 * This source code is licensed under the BSD-style license found in the
 * LICENSE file in the root directory of this source tree.
 */

package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	lhm "github.com/xboshy/linkedhashmap"
)

type message struct {
	id               uint64
	group            uint64
	publishTimestamp int64
	data             string
}

type container struct {
	lastPublishTimestamp int64
	messages             []*message
}

type mapFunctions struct {
	nEvicted uint64
}

func (mf *mapFunctions) ExpiredHandler(key *uint64, value *container) {
	fmt.Printf("EXPIRED | k: %v, v: %v\n", *key, *value)
	mf.nEvicted++
}

func (mf *mapFunctions) CapacityRule(curcapacity uint64, curlen uint64, head *container, tail *container) uint64 {
	/*
	 * lhm capacity can be calculated from head/tail elements
	 * example:
	 *   delta := tail.lastPublishTimestamp - head.lastPublishTimestamp
	 *   // define capacity using the delta
	 */

	calcapacity := curcapacity

	if curlen > 10 {
		calcapacity = calcapacity - 1
	} else {
		calcapacity = calcapacity + 5
	}

	if calcapacity != curcapacity {
		fmt.Printf("CAPACITY | old: %v, new: %v\n", curcapacity, calcapacity)
	}

	return calcapacity
}

func produce(chn chan *message, m *lhm.Map[uint64, container], mf *mapFunctions, mu *sync.Mutex) {
	defer mu.Unlock()
	for {
		msg := <-chn
		if msg == nil {
			break
		}

		ctemp := m.Get(msg.group)
		var c container
		if ctemp == nil {
			c = container{
				lastPublishTimestamp: 0,
				messages:             make([]*message, 0),
			}
		} else {
			c = *ctemp
		}

		if msg.publishTimestamp > c.lastPublishTimestamp {
			c.lastPublishTimestamp = msg.publishTimestamp
		}
		c.messages = append(c.messages, msg)

		m.Push(msg.group, c)
		fmt.Printf("PRODUCED | group: %v, msg: %v\n", msg.group, c)
	}
}

func main() {
	var mu sync.Mutex
	chn := make(chan *message, 10)
	mf := mapFunctions{}
	m := lhm.New[uint64, container](
		0,
		&mf,
	)

	mu.Lock()
	defer mu.Unlock()
	go produce(chn, m, &mf, &mu)

	var i uint64
	for i = 1; i <= 1000; i++ {
		msg := &message{
			id:               i,
			group:            rand.Uint64() % 32,
			publishTimestamp: time.Now().UnixMilli(),
			data:             fmt.Sprintf("Value: %v", i),
		}

		chn <- msg
	}
	chn <- nil // poison pill
	mu.Lock()

	len := m.Len()
	fmt.Printf("LHM LEN: %v\n", len)
}
