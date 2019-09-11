package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	dbPath    string
	counterOn bool
)

var rootCmd = &cobra.Command{
	Use:   "state-tools",
	Short: "state-tools analyses aergo state at a given trie root",
	Long: `state-tools parses every trie node and leaf in the Aergo trie of given root.
		   When parsing the general trie, accounts are analysed to count all account types and balances.
		   Snapshots can be created for the whole state and stored in a new database.
		   Other statistics about blockchain state and chain data size are also provided.`,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&dbPath, "dbPath", "p", "", "Path/to/blockchain/database/folder/data")
	rootCmd.PersistentFlags().BoolVarP(&counterOn, "counterOn", "c", true, "Make a counter of db reads")
	rootCmd.MarkPersistentFlagRequired("dbPath")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
