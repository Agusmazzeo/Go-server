package controllers

import (
	"server/src/scheduler"
	"sync"

	"gorm.io/gorm"
)

type Controller struct {
	DB             *gorm.DB
	SchedulerMutex sync.Mutex
	Schedulers     map[uint]*scheduler.ScheduledTask
}

func NewController(db *gorm.DB) *Controller {
	return &Controller{DB: db, SchedulerMutex: sync.Mutex{}, Schedulers: map[uint]*scheduler.ScheduledTask{}}
}

func (c *Controller) GetSchedulers() map[uint]*scheduler.ScheduledTask {
	return c.Schedulers
}
