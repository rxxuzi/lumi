package main

import (
	"fmt"
	"github.com/rxxuzi/lumi/internal/ui"
)

func main() {
	webui := ui.NewWebUI(9720)
	err := webui.Start()
	if err != nil {
		fmt.Printf("Error starting web server: %v\n", err)
	}
}
