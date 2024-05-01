package main

import (
	"context"
	"golang.org/x/time/rate"
	"log"
	"sync/atomic"
	"time"
)

const projectTotalAmount = 200

var totalProjectsProcessed atomic.Int32

func main() {
	log.SetFlags(0)

	projectIdsChan := make(chan []int)
	done := make(chan bool)
	consumerCount := 5
	// Refill rate of 95 requests per minute and bursts of 5 requests to achieve strict limit of
	// 100 requests per minute.
	// It seems reasonable to use a burst of 5 (bucket size) for 5 threads to enable a quick start.
	limiter := rate.NewLimiter(95.0/60, consumerCount)

	start := time.Now()
	for id := 1; id <= consumerCount; id++ {
		id := id
		go consumer(id, start, projectIdsChan, done, limiter)
	}
	go producer(start, projectIdsChan)

	for id := 1; id <= consumerCount; id++ {
		<-done
		log.Printf("%s Done received (%d)", time.Since(start), id)
	}

	totalProjects := totalProjectsProcessed.Load()
	since := time.Since(start)
	log.Printf("%s Done. Actual rate: %d / %f minutes = %f per minute", since, totalProjects, since.Minutes(), float64(totalProjects)/since.Minutes())
}

func producer(start time.Time, projectIdsChan chan<- []int) {
	log.Println(time.Since(start), "Producer launched")
	for i := 0; i < projectTotalAmount; i += 10 {
		projectIds := make([]int, 0, 10)
		for projectId := i; projectId < i+10; projectId++ {
			projectIds = append(projectIds, projectId)
		}
		projectIdsChan <- projectIds
	}
	close(projectIdsChan)
	log.Println(time.Since(start), "Producer done")
}

func consumer(id int, start time.Time, projectIdsChan <-chan []int, done chan<- bool, limiter *rate.Limiter) {
	log.Printf("%s Consumer %d launched", time.Since(start), id)
	for projects := range projectIdsChan {
		for _, _ = range projects {
			limiter.Wait(context.TODO())
			currentCount := totalProjectsProcessed.Add(1)
			if currentCount < 10 || currentCount%5 == 0 {
				log.Printf("%s ProjectIDs processed: %d", time.Since(start), currentCount)
			}
		}
	}
	done <- true
}
