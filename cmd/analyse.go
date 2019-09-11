package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path"

	"github.com/aergoio/aergo-lib/db"
	"github.com/aergoio/aergo/types"
	"github.com/aergoio/state-tools/stool"
	"github.com/gogo/protobuf/proto"
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
	statePath := path.Join(dbPath, "state")
	// check db path and open db
	if stat, err := os.Stat(dbPath); err != nil || !stat.IsDir() {
		fmt.Println("Invalid database path provided")
		return
	}
	store := db.NewDB(db.BadgerImpl, statePath)

	// Get state root
	var rootBytes []byte
	if len(root) != 0 {
		rootBytes, _ = base58.Decode(root)
	} else {
		// query latest state root in state db
		chainPath := path.Join(dbPath, "chain")
		chainStore := db.NewDB(db.BadgerImpl, chainPath)
		latestKey := []byte("chain.latest")
		blockIdx := chainStore.Get(latestKey)
		if blockIdx == nil || len(blockIdx) == 0 {
			fmt.Println("failed to load latest blockidx")
			return
		}

		//blockNo := types.BlockNoFromBytes(blockIdx)
		blockHash := chainStore.Get(blockIdx)
		blockRaw := chainStore.Get(blockHash)
		if blockRaw == nil || len(blockRaw) == 0 {
			fmt.Println("failed to load latest block data")
			return
		}
		block := types.Block{}
		err := proto.Unmarshal(blockRaw, &block)
		if err != nil {
			fmt.Println("failed to unmarshall block")
			return
		}
		if !bytes.Equal(block.Hash, blockHash) {
			fmt.Println("loaded block doest't have expected hash")
			return
		}
		rootBytes = block.Header.BlocksRootHash
	}

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
