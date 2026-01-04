package main

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

func NewScheduller() *cron.Cron {
	jakartaTime, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		fmt.Println("Failed to use Jakarta time. Using UTC instead.")
		jakartaTime = time.UTC
	}
	fmt.Println("Using time zone:", jakartaTime)
	scheduller := cron.New(cron.WithLocation(jakartaTime))

	return scheduller
}
