// Copyright 2026 OrangeCount contributors
// Licensed under the Apache License, Version 2.0 (the "License");

package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"orangecount/internal/repairguidance"
)

func runHelp(args []string, stdout, stderr io.Writer) int {
	args = normalizeHelpArgs(args)
	fs := flag.NewFlagSet("help", flag.ContinueOnError)
	fs.SetOutput(stderr)
	locale := fs.String("locale", "en", "help display locale (en or zh-CN)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *locale != repairguidance.LocaleEnglish && *locale != repairguidance.LocaleChinese {
		fmt.Fprintf(stderr, "orangecount help: unsupported locale %q\n", *locale)
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "orangecount help: expected a topic such as diagnostics/E-EVAL-UNBALANCED")
		return 2
	}
	topic := strings.TrimSpace(fs.Arg(0))
	if !strings.HasPrefix(topic, "diagnostics/") {
		fmt.Fprintf(stderr, "orangecount help: unknown topic %q\n", topic)
		return 1
	}
	code := strings.TrimPrefix(topic, "diagnostics/")
	guide, ok := repairguidance.Lookup(code, *locale)
	if !ok || guide.Topic != topic {
		if *locale == repairguidance.LocaleChinese {
			fmt.Fprintf(stderr, "orangecount help：找不到本地帮助主题 %q，请检查诊断代码。\n", topic)
		} else {
			fmt.Fprintf(stderr, "orangecount help: topic %q not found\n", topic)
		}
		return 1
	}
	return renderGuide(stdout, guide, *locale)
}

func normalizeHelpArgs(args []string) []string {
	var flags, topics []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--locale" && i+1 < len(args) {
			flags = append(flags, args[i], args[i+1])
			i++
			continue
		}
		if strings.HasPrefix(args[i], "--locale=") {
			flags = append(flags, args[i])
			continue
		}
		topics = append(topics, args[i])
	}
	return append(flags, topics...)
}

func renderGuide(w io.Writer, guide repairguidance.Guide, locale string) int {
	labels := map[string][2]string{
		"topic": {"Topic", "主题"}, "what": {"What happened", "发生了什么"}, "why": {"Why it blocks", "为什么阻塞"},
		"inspect": {"Where to inspect", "去哪里检查"}, "steps": {"Safe checks and changes", "安全检查与修改"},
		"example": {"Generic example", "通用示例"}, "before": {"Before", "修改前"}, "after": {"After", "修改后"},
		"note": {"Note", "说明"}, "revalidate": {"Next step", "下一步"},
	}
	label := func(key string) string {
		value := labels[key]
		if locale == repairguidance.LocaleChinese {
			return value[1]
		}
		return value[0]
	}
	fmt.Fprintf(w, "%s: %s\n\n", label("topic"), guide.Topic)
	fmt.Fprintf(w, "%s\n%s\n\n", label("what"), guide.What)
	fmt.Fprintf(w, "%s\n%s\n\n", label("why"), guide.Why)
	fmt.Fprintf(w, "%s\n", label("inspect"))
	for _, item := range guide.Inspect {
		fmt.Fprintf(w, "- %s\n", item)
	}
	fmt.Fprintf(w, "\n%s\n", label("steps"))
	for _, item := range guide.SafeSteps {
		fmt.Fprintf(w, "- %s\n", item)
	}
	fmt.Fprintf(w, "\n%s\n%s:\n%s\n%s:\n%s\n%s:\n%s\n\n%s\n%s\n", label("example"), label("before"), guide.Example.Before, label("after"), guide.Example.After, label("note"), guide.Example.Note, label("revalidate"), guide.Revalidate)
	return 0
}
