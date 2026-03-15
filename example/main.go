package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/xtdlib/schedule"
)

func main() {
	s, err := schedule.Parse([]byte(`{
		"days": "weekdays",
		"time_range": ["07:00", "22:00"],
		"except_dates": ["2026-01-01"]
	}`))
	if err != nil {
		log.Fatal(err)
	}

	s.Watch()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM)

	fmt.Println("running")
	<-sig
	fmt.Println("shutting down")
}
