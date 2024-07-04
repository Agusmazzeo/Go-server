package scheduler

import (
	"github.com/robfig/cron/v3"
)

type ScheduledTask struct {
	cronID cron.EntryID
	cron   *cron.Cron
	cancel chan struct{}
}

func NewScheduledTask(cronSpec string, taskFunc func()) (*ScheduledTask, error) {
	c := cron.New()
	cancel := make(chan struct{})
	task := &ScheduledTask{
		cron:   c,
		cancel: cancel,
	}

	id, err := c.AddFunc(cronSpec, func() {
		select {
		case <-cancel:
			return
		default:
			taskFunc()
		}
	})
	if err != nil {
		return nil, err
	}

	task.cronID = id
	c.Start()
	return task, nil
}

func (s *ScheduledTask) Cancel() {
	s.cron.Remove(s.cronID)
	close(s.cancel)
}
