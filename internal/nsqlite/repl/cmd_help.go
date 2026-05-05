package repl

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/nsqlite/nsqlite/internal/nsqlite/styled"
)

type dotCmd struct {
	name         string
	autocomplete string
	help         string
	args         string
}

func cmdHelpCommands() []dotCmd {
	cmds := []dotCmd{
		{
			name:         ".count [table_name]",
			autocomplete: ".count",
			help:         "Count the number of rows in a table",
			args:         "table_name (required)",
		},
		{
			name:         ".columns [table_name]",
			autocomplete: ".columns",
			help:         "List all columns in a table",
			args:         "table_name (required)",
		},
		{
			name:         ".stats [minutes]",
			autocomplete: ".stats",
			help:         "Shows the server stats of last specified minutes",
			args:         "minutes (optional, default 5)",
		},

		{name: ".tables", autocomplete: ".tables", help: "List all tables in the database"},
		{name: ".indexes", autocomplete: ".indexes", help: "List all indexes in the database"},
		{
			name:         ".functions",
			autocomplete: ".functions",
			help:         "List all functions in the database",
		},
		{name: ".schema", autocomplete: ".schema", help: "List all schema in the database"},
		{name: ".clear", autocomplete: ".clear", help: "Clear the terminal screen"},
		{name: ".help", autocomplete: ".help", help: "Show the help message"},
		{name: ".quit", autocomplete: ".quit", help: "Exit the application"},
		{name: ".exit", autocomplete: ".exit", help: "Exit the application"},
		{name: "CTRL+c", help: "Exit the application"},
	}

	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].name < cmds[j].name
	})

	return cmds
}

func cmdHelp() {
	fmt.Println("Available commands:")
	cmds := cmdHelpCommands()

	tw := styled.NewTableWriter()
	tw.AppendHeader(table.Row{"Command", "Description", "Arguments"})

	for _, cmd := range cmds {
		tw.AppendRow(table.Row{cmd.name, cmd.help, cmd.args})
	}

	fmt.Println(tw.Render())
}

func cmdHelpCompleter(line string) []string {
	suggestions := []string{
		"SELECT ",
		"SELECT * FROM ",
		"SELECT COUNT(*) FROM ",
		"INSERT INTO ",
		"UPDATE",
		"DELETE FROM ",
		"CREATE TABLE ",
		"DROP TABLE ",
		"ALTER TABLE ",
	}

	for _, cmd := range cmdHelpCommands() {
		if cmd.autocomplete != "" {
			suggestions = append(suggestions, cmd.autocomplete)
		}
	}

	results := []string{}
	for _, suggestion := range suggestions {
		if strings.HasPrefix(strings.ToLower(suggestion), strings.ToLower(line)) {
			results = append(results, suggestion)
		}
	}

	return results
}
