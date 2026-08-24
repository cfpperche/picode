// Debug tool: load PiCode, capture console errors, click the user-menu
// trigger and report whether the popover opens. Run: go run ./cmd/uicheck
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

func main() {
	url := os.Args[1]
	allocCtx, cancelA := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("ignore-certificate-errors", true),
		)...)
	defer cancelA()
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if ce, ok := ev.(*runtime.EventConsoleAPICalled); ok {
			for _, a := range ce.Args {
				fmt.Printf("console[%s]: %s\n", ce.Type, a.Value)
			}
		}
		if ee, ok := ev.(*runtime.EventExceptionThrown); ok && ee.ExceptionDetails != nil {
			fmt.Printf("EXCEPTION: %s | %s\n", ee.ExceptionDetails.Text, ee.ExceptionDetails.Exception.Description)
			if ee.ExceptionDetails.URL != "" {
				fmt.Printf("  at %s:%d\n", ee.ExceptionDetails.URL, ee.ExceptionDetails.LineNumber)
			}
		}
	})

	var hiddenBefore, hiddenAfter, themeNow, hash, popAfterTheme string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`document.getElementById("um-popover").hidden ? "true" : "false"`, &hiddenBefore),
		chromedp.Click("#um-trigger", chromedp.ByID),
		chromedp.Sleep(300*time.Millisecond),
		chromedp.Evaluate(`document.getElementById("um-popover").hidden ? "true" : "false"`, &hiddenAfter),
	)
	if err != nil {
		fmt.Println("run error:", err)
		os.Exit(1)
	}
	fmt.Printf("popover hidden before=%v after=%v\n", hiddenBefore, hiddenAfter)
	if hiddenAfter == "false" {
		fmt.Println("RESULT: popover OPENS ✓")
	} else {
		fmt.Println("RESULT: popover did NOT open ✗")
		os.Exit(1)
	}

	// Functional: theme button inside popover applies the theme and closes it.
	// (popover is still open from the previous sequence)
	err = chromedp.Run(ctx,
		chromedp.Click(`#um-popover [data-theme-option="light"]`, chromedp.ByQuery),
		chromedp.Sleep(400*time.Millisecond),
		chromedp.Evaluate(`document.documentElement.dataset.theme + "|" + document.getElementById("um-popover").hidden + "|" + localStorage.getItem("picode-theme")`, &themeNow),
	)
	if err != nil {
		fmt.Println("theme run error:", err)
		os.Exit(1)
	}
	fmt.Printf("theme after click: %s\n", themeNow)
	ok := themeNow == "light|true|light"
	if ok {
		fmt.Println("RESULT: theme applies from popover + closes ✓")
	} else {
		fmt.Println("RESULT: theme click broken ✗")
		os.Exit(1)
	}

	// Functional: Settings link navigates to #/settings (popover closed after
	// theme pick — reopen it first).
	err = chromedp.Run(ctx,
		chromedp.Click("#um-trigger", chromedp.ByID),
		chromedp.Sleep(150*time.Millisecond),
		chromedp.Click("#um-settings", chromedp.ByID),
		chromedp.Sleep(400*time.Millisecond),
		chromedp.Evaluate(`location.hash + "|" + document.getElementById("settings-view").hidden`, &hash),
	)
	_ = popAfterTheme
	if err != nil {
		fmt.Println("settings run error:", err)
		os.Exit(1)
	}
	fmt.Printf("settings after click: %s\n", hash)
	if hash == "#/settings|false" {
		fmt.Println("RESULT: settings route opens ✓")
	} else {
		fmt.Println("RESULT: settings link broken ✗")
		os.Exit(1)
	}
}
