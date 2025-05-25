package controllers

import (
	"server/src/scheduler"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Controller struct {
	DB             *pgxpool.Pool
	SchedulerMutex sync.Mutex
	Schedulers     map[uint]*scheduler.ScheduledTask
}

func NewController(db *pgxpool.Pool) *Controller {
	return &Controller{DB: db, SchedulerMutex: sync.Mutex{}, Schedulers: map[uint]*scheduler.ScheduledTask{}}
}

func (c *Controller) GetSchedulers() map[uint]*scheduler.ScheduledTask {
	return c.Schedulers
}
