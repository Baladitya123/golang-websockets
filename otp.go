// the otp will send to the client at the first login and again send to the server for the verification to the connection to websockets
package main

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Otp struct {
	key       string
	createdAt time.Time
}

type RetentionMap map[string]Otp

func newRetentionMap(ctx context.Context, retentionPeriod time.Duration) RetentionMap {
	rm := make(RetentionMap)
	go rm.retention(ctx, retentionPeriod)
	return rm
}

func (r RetentionMap) newOtp() Otp {
	o := Otp{
		key:       uuid.NewString(), // create a random unique string
		createdAt: time.Now(),
	}
	r[o.key] = o
	return o
}

func (r RetentionMap) verifyOtp(otp string) bool {
	if _, ok := r[otp]; !ok {
		return false
	}
	delete(r, otp)
	return true
}

func (r RetentionMap) retention(ctx context.Context, retentionPeriod time.Duration) {
	ticker := time.NewTicker(400 * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			for _, otp := range r {
				if otp.createdAt.Add(retentionPeriod).Before(time.Now()) {
					delete(r, otp.key)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
