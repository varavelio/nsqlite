package repl

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/nsqlite/nsqlite/internal/nsqlite/styled"
	"github.com/nsqlite/nsqlite/internal/util/numutil"
)

func cmdStats(r *Repl, statsQty int) {
	stats, err := r.client.GetStats(context.Background())
	if err != nil {
		fmt.Println("Failed to get stats:", err)
		return
	}

	tw := styled.NewTableWriter()
	tw.AppendHeader(
		table.Row{
			"Minute (UTC)",
			"Reads",
			"Writes",
			"Begins",
			"Commits",
			"Rollbacks",
			"Errors",
			"Requests",
		},
	)

	rows := []table.Row{}
	for i, stat := range stats.Stats {
		if i >= statsQty {
			break
		}

		minute, err := time.Parse(time.RFC3339, stat.Minute)
		if err != nil {
			continue
		}

		rows = append(rows, table.Row{
			minute.Format("2006-01-02 15:04"),
			numutil.IntWithCommas(stat.Reads),
			numutil.IntWithCommas(stat.Writes),
			numutil.IntWithCommas(stat.Begins),
			numutil.IntWithCommas(stat.Commits),
			numutil.IntWithCommas(stat.Rollbacks),
			numutil.IntWithCommas(stat.Errors),
			numutil.IntWithCommas(stat.HTTPRequests),
		})
	}
	slices.Reverse(rows)
	tw.AppendRows(rows)

	tw.AppendFooter(table.Row{
		"Total",
		numutil.IntWithCommas(stats.Totals.Reads),
		numutil.IntWithCommas(stats.Totals.Writes),
		numutil.IntWithCommas(stats.Totals.Begins),
		numutil.IntWithCommas(stats.Totals.Commits),
		numutil.IntWithCommas(stats.Totals.Rollbacks),
		numutil.IntWithCommas(stats.Totals.Errors),
		numutil.IntWithCommas(stats.Totals.HTTPRequests),
	})

	fmt.Println(tw.Render())
	styled.DimmedColor().Printf("Showing the last %d minutes of stats\n", statsQty)
	styled.DimmedColor().Printf("Uptime: %s\n", stats.Uptime)
	fmt.Println()
}
