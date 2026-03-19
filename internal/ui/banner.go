package ui

import (
	"fmt"
	"strings"
)

// PrintEnvBanner prints a colored environment banner.
func PrintEnvBanner(env string) {
	label := "ENVIRONMENT: " + strings.ToUpper(env)
	width := len(label) + 4
	border := "╔" + strings.Repeat("═", width) + "╗"
	middle := "║  " + label + "  ║"
	bottom := "╚" + strings.Repeat("═", width) + "╝"

	if noColor {
		fmt.Fprintln(out, border)
		fmt.Fprintln(out, middle)
		fmt.Fprintln(out, bottom)
		return
	}

	var render func(string) string
	switch env {
	case "prod":
		render = func(s string) string { return BannerProdStyle.Render(s) }
	case "stage":
		render = func(s string) string { return BannerStageStyle.Render(s) }
	default:
		render = func(s string) string { return BannerDevStyle.Render(s) }
	}

	fmt.Fprintln(out, render(border))
	fmt.Fprintln(out, render(middle))
	fmt.Fprintln(out, render(bottom))
	fmt.Fprintln(out)
}
