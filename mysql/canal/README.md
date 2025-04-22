# MySQL Canal

A lightweight and easy-to-use Go library to capture and process MySQL binlog events.

## Installation

```bash
go get github.com/go-fries/fries/mysql/canal/v3
```

## Usage

```go
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

	l.sb.WriteString(fmt.Sprintf("Table: %s.%s\n", event.RowsEvent.Table.Schema, event.RowsEvent.Table.Name))
	l.sb.WriteString(fmt.Sprintf("Action: %s\n", event.RowsEvent.Action))
	for i, row := range event.RowsEvent.Rows {
		maps := make(map[string]any, len(row))
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
```

output:

```shell
Table: table.users
Action: update
		Row 1: {"id": 1, "name": "flc"},
		Row 2: {"id": 1, "name": "FLC"},
Time: 2025-04-08 14:42:29
Binlog position: 673094763, EventType: UpdateRowsEventV2
------------------------------------------------------------------------------------

Table: table.articles
Action: insert
		Row 1: {"batch_id":"9e9fdd9d-85f3-4c33-ac85-b69dec687fe0","content":"","created_at":"2025-04-08 14:42:30","family_hash":null,"sequence":396869701,"should_display_on_index":1,"type":"event","uuid":"9e9fdd9d-7dc3-4ca0-8d35-827552d76979"}
		Row 2: {"batch_id":"9e9fdd9d-85f3-4c33-ac85-b69dec687fe0","content":"","created_at":"2025-04-08 14:42:30","family_hash":null,"sequence":396869702,"should_display_on_index":1,"type":"redis","uuid":"9e9fdd9d-7fcd-4162-9214-aedbf7f8a8c5"}
Time: 2025-04-08 14:42:30
Binlog position: 673119073, EventType: WriteRowsEventV2
------------------------------------------------------------------------------------
```