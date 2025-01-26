package nsqlitebench

import (
	"fmt"
	"strconv"

	"github.com/peterh/liner"
)

// benchmarksConfig holds all parameters for each benchmark.
type benchmarksConfig struct {
	benchmarkSimpleConfig
	benchmarkComplexConfig
	benchmarkManyConfig
	benchmarkLargeConfig
}

func promptInt(prompt string, defaultValue int) int {
	line := liner.NewLiner()
	defer line.Close()
	line.SetCtrlCAborts(true)

	for {
		resp, err := line.Prompt(prompt)
		if err != nil {
			continue
		}
		if resp == "" {
			return defaultValue
		}

		i, err := strconv.Atoi(resp)
		if err != nil {
			continue
		}

		if i > 0 {
			return i
		}

		return defaultValue
	}
}

func promptConfig() benchmarksConfig {
	line := liner.NewLiner()
	defer line.Close()
	line.SetCtrlCAborts(true)

	queryGoroutines := promptInt("Read goroutines (default 100): ", 100)
	insertGoroutines := promptInt("Write goroutines (default 100): ", 100)
	fmt.Println()

	return benchmarksConfig{
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
