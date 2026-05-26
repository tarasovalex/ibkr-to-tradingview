package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "ibkr2tv",
		Short: "Convert IBKR Activity Statement CSV to TradingView portfolio CSV",
	}
	root.AddCommand(newConvertCmd())
	return root
}
