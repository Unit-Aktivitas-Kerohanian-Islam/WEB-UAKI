package main

import (
	"log"

	"backend_server/actions"
	_ "backend_server/grifts"
)

func main() {
	app := actions.App()
	if err := app.Serve(); err != nil {
		log.Fatal(err)
	}
}

