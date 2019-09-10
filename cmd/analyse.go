package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/aergoio/aergo-lib/db"
	"github.com/aergoio/state-tools/stool"
	"github.com/mr-tron/base58/base58"
	"github.com/spf13/cobra"
)

var (
	contractTrie bool
)

func init() {
	analyseCmd.Flags().BoolVar(&contractTrie, "contractTrie", false, "The trie being queried is a contract trie")
	rootCmd.AddCommand(analyseCmd)
}

var analyseCmd = &cobra.Command{
	Use:   "analyse",
	Short: "Analyse the database",
	Run:   execAnalyse,
}

func execAnalyse(cmd *cobra.Command, args []string) {
	rootBytes, _ := base58.Decode(root)

	// check db path
	statePath := path.Join(dbPath, "state")
	if stat, err := os.Stat(statePath); err != nil || !stat.IsDir() {
		fmt.Println("Invalid database path provided")
		return
	}

	store := db.NewDB(db.BadgerImpl, statePath)
	sa := stool.NewStateAnalysis(store, counterOn, !contractTrie, 10000)

	err := sa.Analyse(rootBytes)
	if err != nil {
		fmt.Println(err)
		return
	}
	store.Close()

	displayResults(sa, contractTrie)
	displayFolderSizes(dbPath, "Size information:")
}
