package main

import (
    "context"
    "fmt"
    "log"

    "github.com/chromedp/chromedp"
)

func main() {
    // Set options to run Chromium in non-headless mode (visible window)
    opts := append(chromedp.DefaultExecAllocatorOptions[:],
        chromedp.Flag("headless", false), // Disable headless mode
    )

    // Create a new context with custom allocator options
    allocatorCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
    defer cancel()

    // Create a new chromedp context using the allocator context
    ctx, cancel := chromedp.NewContext(allocatorCtx)
    defer cancel()

    // Variable to store the page title
    var title string

    // Run tasks: navigate to a page and retrieve its title
    err := chromedp.Run(ctx,
        chromedp.Navigate("https://www.google.com"),
        chromedp.Title(&title),
    )

    // Handle errors
    if err != nil {
        log.Fatal(err)
    }

    // Print the page title to the console
    fmt.Println("Page title:", title)
}
