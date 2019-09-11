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
	root         string
)

func init() {
	analyseCmd.Flags().BoolVar(&contractTrie, "contractTrie", false, "The trie being queried is a contract trie")
	analyseCmd.Flags().StringVarP(&root, "root", "r", "", "Root of the Aergo trie to analyse/snapshot")
	rootCmd.AddCommand(analyseCmd)
}

var analyseCmd = &cobra.Command{
	Use:   "analyse",
	Short: "Analyse the database",
	Run:   execAnalyse,
}

func execAnalyse(cmd *cobra.Command, args []string) {
	statePath := path.Join(dbPath, "state")
	// check db path and open db
	if stat, err := os.Stat(dbPath); err != nil || !stat.IsDir() {
		fmt.Println("Invalid database path provided")
		return
	}
	store := db.NewDB(db.BadgerImpl, statePath)

	// Get state root
	var rootBytes []byte
	var err error
	if len(root) != 0 {
		rootBytes, err = base58.Decode(root)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		// query latest state root in chain db
		chainPath := path.Join(dbPath, "chain")
		rootBytes, err = getLatestTrieRoot(chainPath)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	sa := stool.NewStateAnalysis(store, counterOn, !contractTrie, 10000)
	err = sa.Analyse(rootBytes)
	if err != nil {
		fmt.Println(err)
		return
	}
	store.Close()

	displayResults(sa, contractTrie)
	displayFolderSizes(dbPath, "Size information:")
}
