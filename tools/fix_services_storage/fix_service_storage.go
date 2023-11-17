package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Filecoin-Titan/titan-container/api/types"
	"github.com/Filecoin-Titan/titan-container/db"
	"log"
	"os"
)

func main() {
	conn, err := db.SqlDB(os.Getenv("DSN"))
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	var oldStorage []string

	query := `SELECT id, storage FROM services`

	rows, err := conn.Queryx(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id         int64
			storageStr string
		)

		if err := rows.Scan(&id, &storageStr); err != nil {
			log.Println(err)
			continue
		}

		var storage types.Storage
		if err := json.Unmarshal([]byte(storageStr), &storage); err != nil {
			log.Println(err)
			continue
		}

		newStorages := []*types.Storage{&storage}
		val, err := json.Marshal(newStorages)
		if err != nil {
			log.Println(err)
			continue
		}

		updateStatement := `update services set storage = ? where id = ?`
		_, err = conn.ExecContext(ctx, updateStatement, val, id)
		if err != nil {
			log.Println(err)
			continue
		}

	}

	fmt.Println(oldStorage)
}
