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
	Short: "state-tools analyses, verifies integrity and creates snapshots of aergo state at a given trie root",
	Long: `state-tools parses every trie node and leaf in the Aergo trie of given root. 
	Functionlity : 
	- Analyse without state integrity: gets information about trie leaves
	- Analyse with state integrity: gets information about trie leaves and also analyses contract storage tries for integrity.
	- Snapshot state (copies the general and contract tries)`,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&dbPath, "dbPath", "p", "", "Path/to/blockchain/database/folder/data")
	rootCmd.PersistentFlags().BoolVarP(&countDBReads, "countDBReads", "c", true, "Make a counter of db reads")
	rootCmd.PersistentFlags().BoolVarP(&integrityCheck, "integrityCheck", "i", true, "Analyse general and all contract trie nodes to check integrity.")
	rootCmd.MarkPersistentFlagRequired("dbPath")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
