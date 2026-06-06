package main

import (
	"context"
	"log"

	"mini-store-go/backend/internal/app"
)

func main() {
	if err := app.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
