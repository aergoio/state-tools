package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/aergoio/aergo-lib/db"
	"github.com/aergoio/aergo/types"
	"github.com/aergoio/state-tools/stool"
	"github.com/mr-tron/base58/base58"
	"github.com/spf13/cobra"
)

var (
	contractTrie bool
	root         string
	blockHeight  uint64
)

func init() {
	analyseCmd.Flags().BoolVar(&contractTrie, "contractTrie", false, "The trie being queried is a contract trie")
	analyseCmd.Flags().StringVarP(&root, "root", "r", "", "Root of the Aergo trie to analyse")
	analyseCmd.Flags().Uint64VarP(&blockHeight, "blockHeight", "b", 0, "Block height to analyse")
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

	if len(root) != 0 && blockHeight != 0 {
		fmt.Println("choose between root and blockHeight flags")
		return
	}
	if contractTrie && len(root) == 0 {
		fmt.Println("must provide storage root for analysing contract trie")
		return
	}

	chainPath := path.Join(dbPath, "chain")
	chainStore := db.NewDB(db.BadgerImpl, chainPath)

	// Get state root
	var rootBytes []byte
	var err error
	if len(root) != 0 {
		rootBytes, err = base58.Decode(root)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else if blockHeight != 0 {
		heightBytes := types.BlockNoToBytes(blockHeight)
		rootBytes, err = getTrieRoot(chainStore, heightBytes)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		// query latest state root in chain db
		rootBytes, err = getLatestTrieRoot(chainStore)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	chainStore.Close()

	fmt.Println("\nAnalysing state with root: ", base58.Encode(rootBytes))
	sa := stool.NewStateAnalysis(store, countDBReads, !contractTrie, integrityCheck, 10000)
	err = sa.Analyse(rootBytes)
	if err != nil {
		fmt.Println(err)
		return
	}
	store.Close()

	displayResults(sa, contractTrie)
	displayFolderSizes(dbPath, "Current latest state size information:")
}
