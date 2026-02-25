package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	positionerredis "github.com/go-fries/fries/mysql/canal/positioner/redis/v3"
	"github.com/go-fries/fries/mysql/canal/v3"
)

type Listener struct {
	sb strings.Builder
}

var _ canal.RowListener = (*Listener)(nil)

func (l *Listener) OnRow(_ context.Context, event *canal.RowEvent) error {
	defer l.sb.Reset()

	fmt.Fprintf(&l.sb, "Table: %s.%s\n", event.RowsEvent.Table.Schema, event.RowsEvent.Table.Name)
	fmt.Fprintf(&l.sb, "Action: %s\n", event.RowsEvent.Action)
	for i, row := range event.RowsEvent.Rows {
		maps := make(map[string]any, len(row))
		for j, col := range event.RowsEvent.Table.Columns {
			maps[col.Name] = row[j]
		}
		bytes, err := json.Marshal(maps)
		if err != nil {
			return err
		}
		fmt.Fprintf(&l.sb, "\t\tRow %d: %s\n", i+1, string(bytes))
	}
	fmt.Fprintf(&l.sb, "Time: %s\n", time.Unix(int64(event.RowsEvent.Header.Timestamp), 0).Format(time.DateTime))
	fmt.Fprintf(&l.sb, "Binlog position: %d, EventType: %s\n", event.RowsEvent.Header.LogPos, event.RowsEvent.Header.EventType)
	l.sb.WriteString("------------------------------------------------------------------------------------\n")
	fmt.Println(l.sb.String())
	return nil
}

func main() {
	c, err := canal.New(&canal.Config{
		Addr:               "localhost:3306",
		User:               "root",
		Password:           "123456",
		IncludeTablesRegex: []string{"test\\..*"},
		ExcludeTablesRegex: []string{".*no.*"},
	},
		canal.WithPositioner(positionerredis.NewBufferedPositioner(nil)), // replace with your Redis client
		canal.WithListeners(&Listener{}),
	)
	if err != nil {
		panic(err)
	}

	time.AfterFunc(1000*time.Second, func() { //nolint:mnd
		_ = c.Stop(context.Background())
	})

	if err := c.Start(context.Background()); err != nil {
		panic(err)
	}
}
