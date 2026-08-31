package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	to := os.Getenv("DEMO_EMAIL_TO")
	if to == "" {
		fmt.Fprintln(os.Stderr, "DEMO_EMAIL_TO is required")
		os.Exit(1)
	}
	client, err := NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	id, sent, err := SendDeliveryUpdate(context.Background(), client, NewSuppressionLog(), to, "PKG-1042")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if sent {
		fmt.Println("delivery update sent:", id)
	} else {
		fmt.Println("delivery update skipped")
	}
}
