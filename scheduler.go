package report_receiver_bot

import (
     "log"
     "time"
     // "container/heap"
)

// simple scheduler implementation
type JobType func() (bool, error)
func Schedule(timeout time.Duration, count int, job JobType) {
    go func() {
        for i := 1; i < count + 1; i++ {
            time.Sleep(timeout * time.Duration(i))
            ok, err := job()
            if err != nil {
                log.Printf("[info] Scheduler error: %v", err)
            }
            if ok {
                return // cancel job
            }
        }
    }()
}

