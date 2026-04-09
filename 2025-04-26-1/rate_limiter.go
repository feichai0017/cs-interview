package main

import (
	"fmt"
	"sync"
	"time"
)

type userData struct {
  timestamps []time.Time
}

type RateLimiter struct {
  mu      sync.Mutex
  users  map[string]*userData
}

func NewRateLimiter() *RateLimiter {
  rl := &RateLimiter{
    users: make(map[string]*userData),
  }
  return rl
}

func (rl *RateLimiter) Allow(userID string) bool {
  rl.mu.Lock()
  defer rl.mu.Unlock()

  now := time.Now()
  user, ok := rl.users[userID]

  if !ok {
    user = &userData{}
    rl.users[userID] = user
  }

  // Remove timestamps older than 1 minute
  validTimestamps := make([]time.Time, 0, len(user.timestamps))
  for _, ts := range user.timestamps {
    if now.Sub(ts) <= time.Minute {
      validTimestamps = append(validTimestamps, ts)
    }
  }
  user.timestamps = validTimestamps

  if len(user.timestamps) >= 100 {
    return false
  }

  user.timestamps = append(user.timestamps, now)
  return true

}

func main() {
  rl := NewRateLimiter()
  userID := "user1"

  // send 100 requests
  for i := 0; i < 100; i++ {
      allowed := rl.Allow(userID)
      if !allowed {
          fmt.Println("Unexpected rejection at", i)
      }
  }

  // send 101st request, should be rejected
  if rl.Allow(userID) {
      fmt.Println("Error: Should have been rejected!")
  } else {
      fmt.Println("Correctly rejected the 101st request")
  }

  // wait for 61 seconds to allow the timestamps to expire
  fmt.Println("Sleeping for 61 seconds...")
  time.Sleep(61 * time.Second)

  if rl.Allow(userID) {
      fmt.Println("Correctly allowed after reset")
  } else {
      fmt.Println("Error: Should have been allowed after reset")
  }
}
