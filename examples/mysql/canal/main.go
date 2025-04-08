package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-fries/fries/mysql/canal/v3"
)

type Listener struct {
	sb strings.Builder
}

var _ canal.RowEventListener = (*Listener)(nil)

func (l *Listener) OnRow(_ context.Context, event *canal.RowEvent) error {
	defer l.sb.Reset()

	l.sb.WriteString(fmt.Sprintf("Table: %s.%s\n", event.RowsEvent.Table.Schema, event.RowsEvent.Table.Name))
	l.sb.WriteString(fmt.Sprintf("Action: %s\n", event.RowsEvent.Action))
	for i, row := range event.RowsEvent.Rows {
		maps := make(map[string]interface{}, len(row))
		for j, col := range event.RowsEvent.Table.Columns {
			maps[col.Name] = row[j]
		}
		bytes, err := json.Marshal(maps)
		if err != nil {
			return err
		}
		l.sb.WriteString(fmt.Sprintf("\t\tRow %d: %s\n", i+1, string(bytes)))
	}
	l.sb.WriteString(fmt.Sprintf("Time: %s\n", time.Unix(int64(event.RowsEvent.Header.Timestamp), 0).Format(time.DateTime)))
	l.sb.WriteString(fmt.Sprintf("Binlog position: %d, EventType: %s\n", event.RowsEvent.Header.LogPos, event.RowsEvent.Header.EventType))
	l.sb.WriteString("------------------------------------------------------------------------------------\n")
	fmt.Println(l.sb.String())
	return nil
}

func main() {
	c, err := canal.New(&canal.Config{
		Addr:     "localhost:3306",
		User:     "root",
		Password: "123456",
	}, canal.WithListeners(&Listener{}))
	if err != nil {
		panic(err)
	}

	time.AfterFunc(1000*time.Second, func() {
		_ = c.Stop(context.Background())
	})

	if err := c.Start(context.Background()); err != nil {
		panic(err)
	}
}
