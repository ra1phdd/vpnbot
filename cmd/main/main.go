package main

import (
	"log"
	"nsvpn/internal/pkg/app"
)

func main() {
	err := app.New()
	if err != nil {
		log.Fatal(err)
	}
}
