package repl

import (
	"context"
	"fmt"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/nsqlite/nsqlite/internal/nsqlite/styled"
	"github.com/nsqlite/nsqlite/internal/nsqlited/db"
	"github.com/nsqlite/nsqlitego/nsqlitehttp"
)

func cmdQuery(r *Repl, input string, params []nsqlitehttp.QueryParam) {
	res, err := r.client.SendQuery(context.TODO(), nsqlitehttp.Query{
		TxID:   r.txID,
		Query:  input,
		Params: params,
	})
	if err != nil && res.Error == "" {
		tw := styled.NewTableWriter()
		tw.AppendHeader(table.Row{"Error"})
		tw.AppendRow(table.Row{err.Error()})
		fmt.Println(tw.Render())
	}

	isError := res.Error != ""
	hasReads := len(res.Columns) > 0
	hasWrites := res.RowsAffected > 0
	hasTxId := res.TxID != ""
	isOk := !isError && !hasReads && !hasWrites

	if isError {
		tw := styled.NewTableWriter()
		tw.AppendHeader(table.Row{"Error"})
		tw.AppendRow(table.Row{r.cleanError(res.Error)})
		fmt.Println(tw.Render())

		if strings.Contains(res.Error, db.ErrTxNotFound.Error()) {
			r.setTxId("")
		}
	}

	if hasTxId {
		tw := styled.NewTableWriter()
		tw.AppendHeader(table.Row{"OK"})
		tw.AppendRow(table.Row{"Transaction started"})
		fmt.Println(tw.Render())
		r.setTxId(res.TxID)
	}

	if isOk {
		tw := styled.NewTableWriter()
		tw.AppendHeader(table.Row{"OK"})
		tw.AppendRow(table.Row{"OK"})
		fmt.Println(tw.Render())
	}

	if hasWrites {
		tw := styled.NewTableWriter()
		tw.AppendHeader(table.Row{"-", "Rows Affected", "Last Insert ID"})
		tw.AppendRow(table.Row{"OK", res.RowsAffected, res.LastInsertID})
		fmt.Println(tw.Render())
	}

	if hasReads {
		tw := styled.NewTableWriter()

		header := table.Row{}
		for _, col := range res.Columns {
			header = append(header, col)
		}
		tw.AppendHeader(header)

		for _, row := range res.Rows {
			tw.AppendRow(row)
		}

		fmt.Println(tw.Render())
	}

	if res.Time > 0 {
		styled.DimmedColor().Printf("Time: %f seconds\n", res.Time)
	}
	fmt.Println()
}
