// Package screenshot captures UI pages to PNG via the Chrome DevTools
// Protocol. It powers the `picode screenshot` subcommand and the
// visual-review loop (see .pi/skills/visual-review).
package screenshot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"
)

// Options configures a capture.
type Options struct {
	URL    string // page to load
	Out    string // destination PNG path (created with parent dirs)
	Width  int    // viewport width (default 1440)
	Height int    // viewport height (default 900)
	Full   bool   // capture full page height instead of viewport
	WaitMS int    // extra settle time after ready (default 500)
}

// Capture renders the page headlessly and writes a PNG to Options.Out.
func Capture(ctx context.Context, opts Options) error {
	if opts.URL == "" {
		return fmt.Errorf("screenshot: url is required")
	}
	if opts.Out == "" {
		return fmt.Errorf("screenshot: out is required")
	}
	if opts.Width <= 0 {
		opts.Width = 1440
	}
	if opts.Height <= 0 {
		opts.Height = 900
	}
	if opts.WaitMS <= 0 {
		opts.WaitMS = 500
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx,
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("disable-gpu", true),
			// PiCode serves HTTPS with a local/self-signed cert (ADR-0007);
			// screenshots are a local capture tool, not a trust decision.
			chromedp.Flag("ignore-certificate-errors", true),
		)...)
	defer cancelAlloc()

	taskCtx, cancelTask := chromedp.NewContext(allocCtx)
	defer cancelTask()

	var buf []byte
	tasks := chromedp.Tasks{
		chromedp.EmulateViewport(int64(opts.Width), int64(opts.Height)),
		chromedp.Navigate(opts.URL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(time.Duration(opts.WaitMS) * time.Millisecond),
	}
	if opts.Full {
		tasks = append(tasks, chromedp.FullScreenshot(&buf, 100))
	} else {
		tasks = append(tasks, chromedp.CaptureScreenshot(&buf))
	}
	if err := chromedp.Run(taskCtx, tasks); err != nil {
		return fmt.Errorf("screenshot: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(opts.Out), 0o755); err != nil {
		return fmt.Errorf("screenshot: %w", err)
	}
	if err := os.WriteFile(opts.Out, buf, 0o644); err != nil {
		return fmt.Errorf("screenshot: %w", err)
	}
	return nil
}
