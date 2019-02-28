package main

import (
	"github.com/valyala/fasthttp"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gocraft/dbr"
	"os"
)

type Event struct {
	At time.Time `db:"at"`
	Name string `db:"name"`
	Value string `db:"value"`
}

func okHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusOK)
}

func bulkInsert(event <- chan Event){
	dataSourceName := os.Getenv("HAKARU_DATASOURCENAME")
	if dataSourceName == "" {
		dataSourceName = "root:hakaru-pass@tcp(127.0.0.1:13306)/hakaru-db"
	}
	db, err := dbr.Open("mysql", dataSourceName, nil)
	if err != nil {
		panic(err.Error())
	}
	events := []Event{}
	for {
		i := <- event
		events = append(events, i)
		if len(events) >= 100 {
			sess := db.NewSession(nil)
			query := sess.InsertInto("eventlog").Columns("at","name", "value")

			for _, value := range events {
				query.Record(value)
			}
			_, err := query.Exec()
			events = events[:0]
			if err != nil {
				panic(err.Error())
			}
		}
	}
}

func main() {

	ch := make(chan Event)
	go bulkInsert(ch)
	hakaruHandler := func(ctx *fasthttp.RequestCtx) {

		name := string(ctx.QueryArgs().Peek("name"))
		value := string(ctx.QueryArgs().Peek("value"))
		ch <- Event{time.Now(),name,value}

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
