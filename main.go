package main

import (
	"fmt"
	"github.com/valyala/fasthttp"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gocraft/dbr"
	"os"
)

type Event struct {
	Name string `db:"name"`
	Value string `db:"value"`
}

func okHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusOK)
}

func bulkInsert(event <- chan Event){
	events := []Event{}
	for {
		i := <- event
		events = append(events, i)
	}
}

func main() {
	dataSourceName := os.Getenv("HAKARU_DATASOURCENAME")
	if dataSourceName == "" {
		dataSourceName = "root:hakaru-pass@tcp(127.0.0.1:13306)/hakaru-db"
	}

	hakaruHandler := func(ctx *fasthttp.RequestCtx) {
		db, err := dbr.Open("mysql", dataSourceName, nil)
		if err != nil {
			panic(err.Error())
		}
		defer db.Close()

		stmt, e := db.Prepare("INSERT INTO eventlog(at, name, value) values(NOW(), ?, ?)")
		if e != nil {
			panic(e.Error())
		}

		defer stmt.Close()

		name := string(ctx.QueryArgs().Peek("name"))
		value := string(ctx.QueryArgs().Peek("value"))

		events = append(events, Event{name, value});

		sess := db.NewSession(nil)
		query := sess.InsertInto("eventlog").Columns("name", "value")

		for _, value := range events {
			query.Record(value)
		}

		if len(events) >= 10 {
			_, err := query.Exec()
			events = events[:0]
			if err != nil {
				panic(err.Error())
			}
		}

		origin := string(ctx.Request.Header.Peek("Origin"))
		if origin != "" {
			ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
			ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
		} else {
			ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
		}
		ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type")
		ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET")
	}


	requestHandler := func(ctx *fasthttp.RequestCtx) {
        // ctx.Path()がnet/httpでいうr.URL.Pathにあたる
        switch string(ctx.Path()) {
        // "/"に対しての処理
        case "/ok":
			okHandler(ctx)
        case "/hakaru":
			hakaruHandler(ctx)
        default:
            ctx.Error("Unsupported path", fasthttp.StatusNotFound)
        }
    }
    // 8080でサーバを起動
	fasthttp.ListenAndServe(":8081", requestHandler)
}
