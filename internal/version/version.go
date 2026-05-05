package version

import (
	"strings"

	"github.com/varavelio/tinta"
)

// These variables are set at build time using ldflags.
var (
	// Version is the vdl version, set at build time using ldflags. This string does not contain the "v" prefix.
	Version = "0.0.0-dev"
	// Commit is the git commit hash, set at build time using ldflags.
	Commit = "unknown"
	// Date is the build date, set at build time using ldflags.
	Date = "unknown"
)

func init() {
	Version = strings.TrimSpace(Version)
	Version = strings.TrimPrefix(Version, "v")
}

// asciiLogo is used to generate AsciiArt.
var asciiLogo = []string{
	`    _   _______ ____    __    _ __     `,
	`   / | / / ___// __ \  / /   (_) /____ `,
	`  /  |/ /\__ \/ / / / / /   / / __/ _ \`,
	` / /|  /___/ / /_/ / / /___/ / /_/  __/`,
	`/_/ |_//____/\___\_\/_____/_/\__/\___/ `,
}

// AsciiArt is the ASCII art for the VDL logo, dynamically generated
// to ensure proper centering and consistent line widths.
var AsciiArt = func() string {
	textBold := tinta.Text().White().Bold()
	textCyan := tinta.Text().Cyan().Bold()
	textGreen := tinta.Text().Green().Bold()
	textLink := textCyan.Underline()

	content := strings.Builder{}

	// Write ascii logo
	for _, line := range asciiLogo {
		content.WriteString(textCyan.String(line))
		content.WriteString("\n")
	}

	// Write version
	content.WriteString(textGreen.String("v" + Version))
	content.WriteString("\n\n")

	// Write basic info
	content.WriteString(
		textBold.String(
			"Star the repo: ",
		) + textLink.String(
			"https://github.com/varavelio/nsqlite",
		),
	)
	content.WriteString("\n")
	content.WriteString(textBold.String("Show usage:    ") + "nsqlite --help")
	content.WriteString("\n")
	content.WriteString(textBold.String("Show version:  ") + "nsqlite --version")

	mainBoxBuilder := tinta.Box().
		Border(tinta.BorderHeavy).
		Blue().
		Bold().
		PaddingX(1)

	// Center logo and version
	for i := range len(asciiLogo) + 1 {
		mainBoxBuilder = mainBoxBuilder.CenterLine(i)
	}

	mainBox := mainBoxBuilder.String(content.String())

	shadowBox := tinta.Box().
		Border(tinta.BorderDouble).
		Blue().
		PaddingX(1).
		String(content.String())

	return tinta.Canvas().Add(shadowBox, 1, 1).Add(mainBox, 0, 0).String()
}()
