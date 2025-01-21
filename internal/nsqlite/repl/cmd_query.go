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

	if res.Type == nsqlitehttp.QueryResponseTypeError {
		tw := styled.NewTableWriter()
		tw.AppendHeader(table.Row{"Error"})
		tw.AppendRow(table.Row{r.cleanError(res.Error)})
		fmt.Println(tw.Render())

		if strings.Contains(res.Error, db.ErrTxNotFound.Error()) {
			r.setTxID("")
		}
		if strings.Contains(res.Error, db.ErrTxNotMatch.Error()) {
			r.setTxID("")
		}
	}

	if res.Type == nsqlitehttp.QueryResponseTypeBegin {
		tw := styled.NewTableWriter()
		tw.AppendHeader(table.Row{"OK"})
		tw.AppendRow(table.Row{"Transaction started"})
		fmt.Println(tw.Render())
		r.setTxID(res.TxID)
	}

	if res.Type == nsqlitehttp.QueryResponseTypeCommit {
		tw := styled.NewTableWriter()
		tw.AppendHeader(table.Row{"OK"})
		tw.AppendRow(table.Row{"Transaction committed"})
		fmt.Println(tw.Render())
		r.setTxID("")
	}

	if res.Type == nsqlitehttp.QueryResponseTypeRollback {
		tw := styled.NewTableWriter()
		tw.AppendHeader(table.Row{"OK"})
		tw.AppendRow(table.Row{"Transaction rolled back"})
		fmt.Println(tw.Render())
		r.setTxID("")
	}

	if res.Type == nsqlitehttp.QueryResponseTypeWrite {
		tw := styled.NewTableWriter()
		tw.AppendHeader(table.Row{"Rows Affected", "Last Insert ID"})
		tw.AppendRow(table.Row{res.RowsAffected, res.LastInsertID})
		fmt.Println(tw.Render())

		if len(res.Rows) > 0 {
			tw = styled.NewTableWriter()
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
	}

	if res.Type == nsqlitehttp.QueryResponseTypeRead {
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
