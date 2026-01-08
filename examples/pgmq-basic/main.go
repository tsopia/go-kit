package main

import (
	"context"
	"log"

	"github.com/tsopia/go-kit/database"
	"github.com/tsopia/go-kit/pgmq"
)

type OrderMessage struct {
	OrderID string `json:"order_id"`
}

func main() {
	db, err := database.New(&database.Config{
		Driver:   "postgres",
		Host:     "127.0.0.1",
		Port:     5432,
		Username: "postgres",
		Password: "password",
		Database: "app",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	adapter, err := pgmq.NewAdapter(context.Background(), db)
	if err != nil {
		log.Fatal(err)
	}

	_, err = pgmq.Configure(adapter)
	if err != nil {
		log.Fatal(err)
	}

	messageID, err := pgmq.SendMsg(context.Background(), "orders", OrderMessage{OrderID: "123"})
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("enqueued message: %d", messageID)
}
