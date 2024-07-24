package main

import (
	"context"
	"fmt"
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

	records := make(map[string]int64)
	var delIds []int64

	query := `select id, name, deployment_id from services order by name, created_at desc;`

	rows, err := conn.Queryx(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id               int64
			name, deployment string
		)

		if err := rows.Scan(&id, &name, &deployment); err != nil {
			log.Println(err)
			continue
		}

		key := fmt.Sprintf("%s_%s", deployment, name)
		_, ok := records[key]
		if !ok {
			records[key] = id
			continue
		}

		delIds = append(delIds, id)

	}

	delsql := `delete from services where id = ?`
	for _, id := range delIds {
		_, err := conn.ExecContext(ctx, delsql, id)
		if err != nil {
			log.Println("del", err)
		}
	}

	fmt.Println("delete", len(delIds))

	log.Println("Success")
}
