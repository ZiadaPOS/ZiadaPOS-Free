// Package scheduler wraps robfig/cron with the Tanzania timezone so apps can
// register recurring jobs (daily reports, credit reminders, cleanup) from
// anywhere.
package scheduler

import (
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler holds a cron instance running in Africa/Dar_es_Salaam.
type Scheduler struct {
	cron   *cron.Cron
	loc    *time.Location
	started bool
}

// New creates a scheduler in the given IANA timezone (default EAT).
func New(tz string) *Scheduler {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.FixedZone("EAT", 3*60*60)
	}
	return &Scheduler{
		cron: cron.New(cron.WithLocation(loc), cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger))),
		loc:  loc,
	}
}

// AddDaily registers a job that runs daily at the given hour/minute.
func (s *Scheduler) AddDaily(hour, minute int, name string, fn func()) error {
	spec := cronSpec(hour, minute)
	_, err := s.cron.AddFunc(spec, func() {
		log.Printf("scheduler: running %q", name)
		fn()
	})
	return err
}

// AddSpec registers a job with a raw cron expression.
func (s *Scheduler) AddSpec(spec, name string, fn func()) (cron.EntryID, error) {
	return s.cron.AddFunc(spec, func() {
		log.Printf("scheduler: running %q", name)
		fn()
	})
}

// Start launches the cron daemon.
func (s *Scheduler) Start() {
	if s.started {
		return
	}
	s.cron.Start()
	s.started = true
	log.Printf("scheduler: started in %s", s.loc)
}

// Stop halts the scheduler.
func (s *Scheduler) Stop() {
	if s.started {
		s.cron.Stop()
		s.started = false
	}
}

func cronSpec(hour, minute int) string {
	return fmt.Sprintf("%d %d * * *", minute%60, hour%24)
}
