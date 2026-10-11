package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

var header = []string{"序号", "平台名称", "音乐名称", "时长", "作者", "音质"}

var rows = [][]string{
	{"1", "kw", "晴天", "04:29", "周杰伦", "320k"},
	{"2", "kg", "晴天", "05:00", "周杰伦", "flac"},
	{"3", "wy", "晴天(原版)", "04:29", "周杰伦", "128k"},
	{"4", "tx", "晴天 (伴奏版) 加长", "10:22", "周杰伦", "flac24bit"},
}

// widths are the *declared* column widths; content area is width-2.
var widths = []int{6, 12, 26, 8, 12, 12}

func render(label string, style tw.BorderStyle, withWidths bool) {
	var b bytes.Buffer
	opts := []tablewriter.Option{
		tablewriter.WithHeader(header),
		tablewriter.WithHeaderAlignment(tw.AlignCenter),
		tablewriter.WithAlignment(tw.Alignment{
			tw.AlignRight, tw.AlignLeft, tw.AlignLeft, tw.AlignRight, tw.AlignLeft, tw.AlignLeft,
		}),
		tablewriter.WithEastAsian(tw.Success),
		tablewriter.WithSymbols(tw.NewSymbols(style)),
	}
	if withWidths {
		m := tw.Mapper[int, int]{}
		for i, w := range widths {
			m[i] = w
		}
		opts = append(opts, tablewriter.WithColumnWidths(m))
	}
	t := tablewriter.NewTable(&b, opts...)
	for _, r := range rows {
		_ = t.Append(r)
	}
	if err := t.Render(); err != nil {
		fmt.Println("render error:", err)
	}
	sum := 0
	for _, w := range widths {
		sum += w
	}
	fmt.Println(strings.Repeat("-", 90))
	fmt.Printf("%-14s sum=%d  widths=%v\n", label, sum, withWidths)
	fmt.Println(strings.Repeat("=", 90))
	for i, ln := range strings.Split(strings.TrimRight(b.String(), "\n"), "\n") {
		fmt.Printf("L%02d w=%3d |%s|\n", i, runewidth.StringWidth(ln), ln)
	}
}

func main() {
	for _, s := range []struct {
		label string
		style tw.BorderStyle
	}{
		{"ASCII", tw.StyleASCII},
		{"None", tw.StyleNone},
		{"Light", tw.StyleLight},
		{"Default", tw.StyleDefault},
		{"Double", tw.StyleDouble},
		{"Heavy", tw.StyleHeavy},
		{"Rounded", tw.StyleRounded},
		{"Markdown", tw.StyleMarkdown},
	} {
		render(s.label, s.style, true)
	}
	fmt.Println("### SAME, AUTO-WIDTH ###")
	render("ASCII-auto", tw.StyleASCII, false)
	render("None-auto", tw.StyleNone, false)
}
