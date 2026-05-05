package nsqlitebench

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/peterh/liner"
)

// benchmarksConfig holds all parameters for each benchmark.
type benchmarksConfig struct {
	execBenchmark bool
	benchmarkSimpleConfig
	benchmarkComplexConfig
	benchmarkManyConfig
	benchmarkLargeConfig
}

func promptBool(prompt string) bool {
	line := liner.NewLiner()
	defer line.Close()
	line.SetCtrlCAborts(true)

	for {
		resp, err := line.Prompt(prompt)
		if err != nil {
			continue
		}
		if resp == "" {
			continue
		}

		resp = strings.ToLower(resp)
		if resp == "y" || resp == "yes" || resp == "true" || resp == "1" {
			return true
		}
		if resp == "n" || resp == "no" || resp == "false" || resp == "0" {
			return false
		}
	}
}

func promptInt(prompt string) int {
	line := liner.NewLiner()
	defer line.Close()
	line.SetCtrlCAborts(true)

	for {
		resp, err := line.Prompt(prompt)
		if err != nil {
			continue
		}
		if resp == "" {
			continue
		}

		i, err := strconv.Atoi(resp)
		if err != nil {
			continue
		}

		if i > 0 {
			return i
		}
	}
}

func promptConfig(ciMode, useRoutines bool) benchmarksConfig {
	queryGoroutines := 1
	insertGoroutines := 1
	if useRoutines {
		queryGoroutines = 100
		insertGoroutines = 100
	}

	if !ciMode {
		line := liner.NewLiner()
		defer line.Close()
		line.SetCtrlCAborts(true)

		execBenchmark := promptBool("Execute benchmark? (y/n): ")
		if !execBenchmark {
			return benchmarksConfig{}
		}

		queryGoroutines = promptInt("Read goroutines: ")
		insertGoroutines = promptInt("Write goroutines: ")
		fmt.Println()
	}

	return benchmarksConfig{
		execBenchmark: true,

		benchmarkSimpleConfig: benchmarkSimpleConfig{
			insertXUsers:     100_000,
			queryYUsers:      200_000,
			insertGoroutines: insertGoroutines,
			queryGoroutines:  queryGoroutines,
		},

		benchmarkComplexConfig: benchmarkComplexConfig{
			insertXUsers:              400,
			insertYArticlesPerUser:    100,
			insertZCommentsPerArticle: 2,
			insertGoroutines:          insertGoroutines,
		},

		benchmarkManyConfig: benchmarkManyConfig{
			insertXUsers:     1_000,
			queryUsersYTimes: 1_000,
			insertGoroutines: insertGoroutines,
			queryGoroutines:  queryGoroutines,
		},

		benchmarkLargeConfig: benchmarkLargeConfig{
			insertXUsers:     10_000,
			insertYBytes:     10_000,
			insertGoroutines: insertGoroutines,
		},
	}
}
