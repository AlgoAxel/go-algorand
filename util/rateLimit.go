// Copyright (C) 2019-2022 Algorand, Inc.
// This file is part of go-algorand
//
// go-algorand is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// go-algorand is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with go-algorand.  If not, see <https://www.gnu.org/licenses/>.

package util

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/algorand/go-algorand/util/metrics"
	"github.com/algorand/go-deadlock"
)

// ElasticRateLimiter holds and distributes capacity through capacityQueues
// Capacity consumers are given an error if there is no capacity available for them,
// and a "capacityGuard" structure they can use to return the capacity when finished
type ElasticRateLimiter struct {
	MaxCapacity            int
	CapacityPerReservation int
	sharedCapacity         capacityQueue
	capacityByClient       map[ErlClient]capacityQueue
	clientLock             *deadlock.RWMutex
	noCapacityCounter      *metrics.Counter
	// CongestionManager and enable flag
	cm                       CongestionManager
	enableCM                 bool
	congestionControlCounter *metrics.Counter
}

// ErlClient clients must support OnClose for reservation closing
type ErlClient interface {
	OnClose(func())
}

// capacity is an empty structure used for loading and draining queues
type capacity struct {
}

// Capacity Queue wraps and maintains a channel of opaque capacity structs
type capacityQueue chan capacity

// ErlCapacityGuard is the structure returned to clients so they can release the capacity when needed
// they also inform the congestion manager of events
type ErlCapacityGuard struct {
	client ErlClient
	cq     capacityQueue
	cm     *CongestionManager
}

// Release will put capacity back into the queue attached to this capacity guard
func (cg ErlCapacityGuard) Release() error {
	select {
	case cg.cq <- capacity{}:
		return nil
	default:
		return fmt.Errorf("could not replace capacity to channel: %v", cg.cq)
	}
}

// Served will notify the CongestionManager that this resource has been served, informing the Service Rate
func (cg ErlCapacityGuard) Served() {
	if *cg.cm != nil {
		(*cg.cm).Served(time.Now())
	}
}

func (q capacityQueue) blockingRelease() {
	q <- capacity{}
}

func (q capacityQueue) blockingConsume() {
	<-q
}

func (q capacityQueue) consume(cm *CongestionManager) (ErlCapacityGuard, error) {
	select {
	case <-q:
		return ErlCapacityGuard{
			cq: q,
			cm: cm,
		}, nil
	default:
		return ErlCapacityGuard{}, fmt.Errorf("could not consume capacity from capacityQueue: %v", q)
	}
}

// NewElasticRateLimiter creates an ElasticRateLimiter and initializes maps
func NewElasticRateLimiter(
	maxCapacity int,
	reservedCapacity int,
	cm CongestionManager,
	nocapCount *metrics.Counter,
	conmanCount *metrics.Counter) *ElasticRateLimiter {
	ret := ElasticRateLimiter{
		MaxCapacity:              maxCapacity,
		CapacityPerReservation:   reservedCapacity,
		capacityByClient:         map[ErlClient]capacityQueue{},
		clientLock:               &deadlock.RWMutex{},
		cm:                       cm,
		sharedCapacity:           capacityQueue(make(chan capacity, maxCapacity)),
		noCapacityCounter:        nocapCount,
		congestionControlCounter: conmanCount,
	}
	// fill the sharedCapacity
	for i := 0; i < maxCapacity; i++ {
		ret.sharedCapacity.blockingRelease()
	}
	return &ret
}

// EnableCongestionControl turns on the flag that the ERL uses to check with its CongestionManager
func (erl *ElasticRateLimiter) EnableCongestionControl() {
	erl.clientLock.Lock()
	erl.enableCM = true
	erl.clientLock.Unlock()
}

// DisableCongestionControl turns off the flag that the ERL uses to check with its CongestionManager
func (erl *ElasticRateLimiter) DisableCongestionControl() {
	erl.clientLock.Lock()
	erl.enableCM = false
	erl.clientLock.Unlock()
}

// ConsumeCapacity will dispense one capacity from either the resource's reservedCapacity,
// and will return a guard who can return capacity when the client is ready
func (erl *ElasticRateLimiter) ConsumeCapacity(c ErlClient) (ErlCapacityGuard, error) {
	var q capacityQueue
	var err error
	var exists bool
	var enableCM bool
	// get the client's queue
	erl.clientLock.RLock()
	q, exists = erl.capacityByClient[c]
	enableCM = erl.enableCM
	erl.clientLock.RUnlock()

	// Step 0: Check for, and create a capacity reservation if needed
	if !exists {
		q, err = erl.openReservation(c)
		if err != nil {
			return ErlCapacityGuard{}, err
		}
		// if the client has been given a new reservation, make sure it cleans up OnClose
		c.OnClose(func() { erl.closeReservation(c) })

		// if this reservation is newly created, directly (blocking) take a capacity
		q.blockingConsume()
		fmt.Println(len(q))
		return ErlCapacityGuard{cq: q, cm: &erl.cm}, nil
	}

	// Step 1: Attempt consumption from the reserved queue
	cg, err := q.consume(&erl.cm)
	if err == nil {
		if erl.cm != nil {
			erl.cm.Consumed(c, time.Now()) // notify the congestion manager that this client consumed from this queue
		}
		return cg, nil
	}
	// Step 2: Potentially gate shared queue access if the congestion manager disallows it
	if erl.cm != nil &&
		enableCM &&
		erl.cm.ShouldDrop(c) {
		if erl.congestionControlCounter != nil {
			erl.congestionControlCounter.Inc(nil)
		}
		return ErlCapacityGuard{}, fmt.Errorf("congestionManager prevented client from consuming capacity")
	}
	// Step 3: Attempt consumption from the shared queue
	cg, err = erl.sharedCapacity.consume(&erl.cm)
	if err != nil {
		if erl.noCapacityCounter != nil {
			erl.noCapacityCounter.Inc(nil)
		}
		return ErlCapacityGuard{}, err
	}
	if erl.cm != nil {
		erl.cm.Consumed(c, time.Now()) // notify the congestion manager that this client consumed from this queue
	}
	return cg, nil
}

// openReservation creates an entry in the ElasticRateLimiter's reservedCapacity map,
// and optimistically transfers capacity from the sharedCapacity to the reservedCapacity
func (erl ElasticRateLimiter) openReservation(c ErlClient) (capacityQueue, error) {
	erl.clientLock.Lock()
	if _, exists := erl.capacityByClient[c]; exists {
		erl.clientLock.Unlock()
		return capacityQueue(nil), fmt.Errorf("client already has a reservation")
	}
	// guard against overprovisioning, if there is less than a reservedCapacity amount left
	remaining := erl.MaxCapacity - (erl.CapacityPerReservation * len(erl.capacityByClient))
	if erl.CapacityPerReservation > remaining {
		erl.clientLock.Unlock()
		return capacityQueue(nil), fmt.Errorf("not enough capacity to reserve for client: %d remaining, %d requested", remaining, erl.CapacityPerReservation)
	}
	// make capacity for the provided client
	q := capacityQueue(make(chan capacity, erl.CapacityPerReservation))
	erl.capacityByClient[c] = q
	erl.clientLock.Unlock()

	// create a thread to drain the capacity from sharedCapacity in a blocking way
	// and move it to the reservation, also in a blocking way
	go func() {
		for i := 0; i < erl.CapacityPerReservation; i++ {
			erl.sharedCapacity.blockingConsume()
			q.blockingRelease()
		}
	}()
	return q, nil
}

// closeReservation will remove the client mapping to capacity channel,
// and will kick off a routine to drain the capacity and replace it to the shared capacity
func (erl ElasticRateLimiter) closeReservation(c ErlClient) {
	erl.clientLock.Lock()
	q, exists := erl.capacityByClient[c]
	// guard clauses, and preventing the ElasticRateLimiter from draining its own sharedCapacity
	if !exists || q == erl.sharedCapacity {
		erl.clientLock.Unlock()
		return
	}
	delete(erl.capacityByClient, c)
	erl.clientLock.Unlock()

	// start a routine to consume capacity from the closed reservation, and return it to the sharedCapacity
	go func() {
		for i := 0; i < erl.CapacityPerReservation; i++ {
			q.blockingConsume()
			erl.sharedCapacity.blockingRelease()
		}
	}()
}

// CongestionManager is an interface for tracking events which happen to capacityQueues
type CongestionManager interface {
	Start(ctx context.Context, wg *sync.WaitGroup)
	Consumed(c ErlClient, t time.Time)
	Served(t time.Time)
	ShouldDrop(c ErlClient) bool
}

type event struct {
	c ErlClient
	t time.Time
}

type shouldDropQuery struct {
	c   ErlClient
	ret chan bool
}

// "Random Early Detection" congestion manager,
// will propose to drop messages proportional to the caller's request rate vs Average Service Rate
type redCongestionManager struct {
	runLock                *deadlock.Mutex
	running                bool
	window                 time.Duration
	consumed               chan event
	served                 chan event
	shouldDropQueries      chan shouldDropQuery
	targetRate             float64
	targetRateRefreshTicks int
	// exp is applied as an exponential factor in shouldDrop. 1 would be linearly proportional, higher values punish noisy neighbors more
	exp float64
	// consumed is the only value tracked by-queue. The others are calculated in-total
	// TODO: If we desire later, we can add mappings onto release/done for more insight
	consumedByClient map[ErlClient]*[]time.Time
	serves           []time.Time
}

// NewREDCongestionManager creates a Congestion Manager which will watches capacityGuard activity,
// and regularly calculates a Target Service Rate, and can give "Should Drop" suggestions
func NewREDCongestionManager(d time.Duration, r int) *redCongestionManager {
	ret := redCongestionManager{
		runLock:                &deadlock.Mutex{},
		window:                 d,
		consumed:               make(chan event, 100000),
		served:                 make(chan event, 100000),
		shouldDropQueries:      make(chan shouldDropQuery, 100000),
		targetRateRefreshTicks: r,
		consumedByClient:       map[ErlClient]*[]time.Time{},
		exp:                    4,
	}
	return &ret
}

// Consumed implements CongestionManager by putting an event on the consumed channel,
// to be processed by the Start() loop
func (cm redCongestionManager) Consumed(c ErlClient, t time.Time) {
	select {
	case cm.consumed <- event{
		c: c,
		t: t,
	}:
	default:
	}
}

// Served implements CongestionManager by putting an event on the done channel,
// to be processed by the Start() loop
func (cm redCongestionManager) Served(t time.Time) {
	select {
	case cm.served <- event{
		t: t,
	}:
	default:
	}
}

// ShouldDrop implements CongestionManager by putting a query shouldDropQueries channel,
// and blocks on the response to return synchronously to the caller
// if an error should prevent the query from running, the result is defaulted to false
func (cm redCongestionManager) ShouldDrop(c ErlClient) bool {
	ret := make(chan bool)
	select {
	case cm.shouldDropQueries <- shouldDropQuery{
		c:   c,
		ret: ret,
	}:
		return <-ret
	default:
		return false
	}
}

// Start will kick off a goroutine to consume activity from the different activity channels,
// as well as service queries about if a given capacityQueue should drop
func (cm *redCongestionManager) Start(ctx context.Context, wg *sync.WaitGroup) {
	// check if the maintainer is already running to ensure there is only one routine
	cm.runLock.Lock()
	defer cm.runLock.Unlock()
	if cm.running {
		return
	}
	cm.running = true
	go cm.run(ctx, wg)
}

func (cm *redCongestionManager) run(ctx context.Context, wg *sync.WaitGroup) {
	tick := 0
	targetRate := float64(0)
	consumedByClient := map[ErlClient]*[]time.Time{}
	serves := []time.Time{}
	exit := false
	for {
		// first process any new events happening
		select {
		// consumed events -- a client has consumed capacity from a queue
		case e := <-cm.consumed:
			if consumedByClient[e.c] == nil {
				ts := []time.Time{}
				consumedByClient[e.c] = &ts
			}
			*(consumedByClient[e.c]) = append(*(consumedByClient[e.c]), e.t)
		// served events -- the capacity has been totally served
		case e := <-cm.served:
			serves = append(serves, e.t)
		// "should drop" queries
		case query := <-cm.shouldDropQueries:
			cutoff := time.Now().Add(-1 * cm.window)
			prune(consumedByClient[query.c], cutoff)
			query.ret <- cm.shouldDrop(targetRate, query.c, consumedByClient[query.c])

		// check for context Done, and start the thread shutdown
		case <-ctx.Done():
			exit = true
		}
		tick = (tick + 1) % cm.targetRateRefreshTicks
		// only recalculate the service rate every N ticks, because all lists must be pruned, which can be expensive
		// also calculate if the routine is going to exit
		if tick == 0 || exit {
			cutoff := time.Now().Add(-1 * cm.window)
			prune(&serves, cutoff)
			for c := range consumedByClient {
				if prune(consumedByClient[c], cutoff) == 0 {
					delete(consumedByClient, c)
				}
			}
			targetRate = 0
			// targetRate is the average service rate per client per second
			if len(consumedByClient) > 0 {
				serviceRate := float64(len(serves)) / float64(cm.window/time.Second)
				targetRate = serviceRate / float64(len(consumedByClient))
			}
		}
		if exit {
			cm.setTargetRate(targetRate)
			cm.setConsumedByClient(consumedByClient)
			cm.setServes(serves)
			cm.stop(wg)
			return
		}
	}
}

func (cm *redCongestionManager) setTargetRate(tr float64) {
	cm.targetRate = tr
}

func (cm *redCongestionManager) setConsumedByClient(cbc map[ErlClient]*[]time.Time) {
	cm.consumedByClient = cbc
}

func (cm *redCongestionManager) setServes(ts []time.Time) {
	cm.serves = ts
}

func (cm *redCongestionManager) stop(wg *sync.WaitGroup) {
	cm.runLock.Lock()
	defer cm.runLock.Unlock()
	cm.running = false
	if wg != nil {
		wg.Done()
	}
}

func prune(ts *[]time.Time, cutoff time.Time) int {
	if ts == nil {
		return 0
	}
	// find the first inserted timestamp *after* the cutoff, and cut everything behind it off
	for i, t := range *ts {
		if t.After(cutoff) {
			*ts = (*ts)[i:]
			return len(*ts)
		}
	}
	//if there are no values after the cutoff, just set to empty and return
	*ts = (*ts)[:0]
	return 0
}

func (cm redCongestionManager) arrivalRateFor(arrivals *[]time.Time) float64 {
	clientArrivalRate := float64(0)
	if arrivals != nil {
		clientArrivalRate = float64(len(*arrivals)) / float64(cm.window/time.Second)
	}
	return clientArrivalRate
}

func (cm redCongestionManager) shouldDrop(targetRate float64, c ErlClient, arrivals *[]time.Time) bool {
	// clients who have "never" been seen do not get dropped
	clientArrivalRate := cm.arrivalRateFor(arrivals)
	if clientArrivalRate == 0 {
		return false
	}
	// A random float is selected, and the arrival rate of the given client is
	// turned to a ratio against targetRate. the congestion manager recommends to drop activity
	// proportional to its overuse above the targetRate
	r := rand.Float64()
	return (math.Pow(clientArrivalRate, cm.exp) / math.Pow(targetRate, cm.exp)) > r
}
