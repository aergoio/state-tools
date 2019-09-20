package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	dbPath         string
	countDBReads   bool
	integrityCheck bool
)

var rootCmd = &cobra.Command{
	Use:   "state-tools",
	Short: "state-tools analyses and creates snapshots of aergo state at a given trie root",
	Long: `state-tools parses every trie node and leaf in the Aergo trie of given root.
		   When parsing the general trie, accounts are analysed to count all account types and balances.
		   Snapshots can be created for the whole state and stored in a new database.
		   Other statistics about blockchain state and chain data size are also provided.`,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&dbPath, "dbPath", "p", "", "Path/to/blockchain/database/folder/data")
	rootCmd.PersistentFlags().BoolVarP(&countDBReads, "countDBReads", "c", true, "Make a counter of db reads")
	rootCmd.PersistentFlags().BoolVarP(&integrityCheck, "integrityCheck", "i", true, "Hash trie nodes to check integrity")
	rootCmd.MarkPersistentFlagRequired("dbPath")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
