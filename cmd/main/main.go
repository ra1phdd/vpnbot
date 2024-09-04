package main

import (
	"log"
	"nsvpn/internal/pkg/app"
)

func main() {
	_, err := app.New()
	if err != nil {
		log.Fatal(err)
	}
}
