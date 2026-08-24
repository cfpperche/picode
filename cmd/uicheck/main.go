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

	var hiddenBefore, hiddenAfter bool
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`document.getElementById("um-popover").hidden`, &hiddenBefore),
		chromedp.Click("#um-trigger", chromedp.ByID),
		chromedp.Sleep(300*time.Millisecond),
		chromedp.Evaluate(`document.getElementById("um-popover").hidden`, &hiddenAfter),
	)
	if err != nil {
		fmt.Println("run error:", err)
		os.Exit(1)
	}
	fmt.Printf("popover hidden before=%v after=%v\n", hiddenBefore, hiddenAfter)
	if !hiddenAfter {
		fmt.Println("RESULT: popover OPENS ✓")
	} else {
		fmt.Println("RESULT: popover did NOT open ✗")
	}
}
