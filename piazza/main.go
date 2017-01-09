package main

import (
	"flag"
	"log"

	piazza "github.com/d4l3k/piazza-api"
)

var (
	username = flag.String("username", "", "Piazza username")
	password = flag.String("password", "", "Piazza password")
)

func main() {
	flag.Parse()

	c, err := piazza.MakeClient(*username, *password)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := c.OptOutOfEmails(); err != nil {
		log.Fatalf("%+v", err)
	}
}
