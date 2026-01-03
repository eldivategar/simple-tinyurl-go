package main

import (
	"time"

	"github.com/robfig/cron/v3"
)

func NewScheduller() *cron.Cron {
	jakartaTime, _ := time.LoadLocation("Asia/Jakarta")
	scheduller := cron.New(cron.WithLocation(jakartaTime))

	return scheduller
}
