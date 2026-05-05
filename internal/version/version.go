package version

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

const Version = "v0.1.0"

// asciiArtTpl returns the ASCII art of nsqlited.
func asciiArtTpl() string {
	lines := []string{
		`    _   _______ ____    __    _ __`,
		`   / | / / ___// __ \  / /   (_) /____`,
		`  /  |/ /\__ \/ / / / / /   / / __/ _ \`,
		` / /|  /___/ / /_/ / / /___/ / /_/  __/`,
		`/_/ |_//____/\___\_\/_____/_/\__/\___/`,
		`%s ` + Version,
		`For more information visit https://github.com/varavelio/nsqlite and please leave a star`,
	}

	lines[0] = color.RGB(214, 245, 245).Sprint(lines[0])
	lines[1] = color.RGB(214, 245, 245).Sprint(lines[1])
	lines[2] = color.RGB(173, 235, 235).Sprint(lines[2])
	lines[3] = color.RGB(132, 225, 225).Sprint(lines[3])
	lines[4] = color.RGB(97, 214, 214).Sprint(lines[4])
	lines[5] = color.RGB(97, 214, 214).Sprint(lines[5])
	lines[6] = color.RGB(97, 214, 214).Sprint(lines[6])

	asciiArt := strings.Join(lines, "\n")
	return asciiArt
}

// ServerVersion returns the server version of nsqlited.
func ServerVersion() string {
	return fmt.Sprintf(asciiArtTpl(), "Server")
}

// CLIVersion returns the CLI version of nsqlite.
func CLIVersion() string {
	return fmt.Sprintf(asciiArtTpl(), "CLI")
}

// BenchVersion returns the benchmark version of nsqlite.
func BenchVersion() string {
	return fmt.Sprintf(asciiArtTpl(), "Benchmark")
}
