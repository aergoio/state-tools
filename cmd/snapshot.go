package cmd

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/aergoio/aergo-lib/db"
	"github.com/aergoio/state-tools/stool"
	sha256 "github.com/minio/sha256-simd"
	"github.com/spf13/cobra"
)

var (
	snapshotPath string
)

func init() {
	snapshotCmd.Flags().StringVarP(&snapshotPath, "snapshotPath", "s", "", "Path/to/a/new/empty/folder/data")
	snapshotCmd.MarkFlagRequired("snapshotPath")
	rootCmd.AddCommand(snapshotCmd)
}

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Create a snapshot of the database",
	Run:   execSnapshot,
}

func execSnapshot(cmd *cobra.Command, args []string) {
	// check db path
	if stat, err := os.Stat(dbPath); err != nil || !stat.IsDir() {
		fmt.Println("Invalid database path provided")
		return
	}
	// check empty snapshot db path
	if stat, err := os.Stat(snapshotPath); err != nil || !stat.IsDir() {
		fmt.Println("Invalid path for snapshot database provided")
		return
	}
	if !isEmpty(snapshotPath) {
		fmt.Println("Snapshot folder must be empty")
		return
	}
	statePath := path.Join(dbPath, "state")
	chainPath := path.Join(dbPath, "chain")
	sqlPath := path.Join(dbPath, "statesql")
	snapshotStatePath := path.Join(snapshotPath, "state")
	snapshotChainPath := path.Join(snapshotPath, "chain")
	snapshotSqlPath := path.Join(snapshotPath, "statesql")

	chainStore := db.NewDB(db.BadgerImpl, chainPath)
	// query latest state root in chain db
	lastRootBytes, err := getLatestTrieRoot(chainStore)
	if err != nil {
		fmt.Println(err)
		return
	}
	// query last vote root trie root
	// it is necessary to snapshot that trie because the dpos will query votes there
	voteRootBytes1, voteRootBytes2, err := getVoteTrieRoots(chainStore)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = os.MkdirAll(snapshotStatePath, 0755)
	if err != nil {
		fmt.Println("Enable to create snapshot state folder")
		return
	}

	store := db.NewDB(db.BadgerImpl, statePath)
	snapshotStore := db.NewDB(db.BadgerImpl, snapshotStatePath)

	// snapshot last state
	fmt.Println("Iterating the Aergo state trie to create snapshot...")
	start := time.Now()
	sa := stool.NewStateAnalysis(store, countDBReads, true, integrityCheck, 10000)
	err = sa.Snapshot(snapshotStore, lastRootBytes)
	if err != nil {
		fmt.Println(err)
		return
	}
	// snapshot last vote states
	hasher := sha256.New()
	hasher.Write([]byte("aergo.system"))
	votingContract := hasher.Sum(nil)
	sva := stool.NewStateAnalysis(store, false, true, integrityCheck, 10000)
	err = sva.SnapshotAccount(snapshotStore, voteRootBytes1, votingContract)
	if err != nil {
		fmt.Println(err)
		return
	}
	sva = stool.NewStateAnalysis(store, false, true, integrityCheck, 10000)
	err = sva.SnapshotAccount(snapshotStore, voteRootBytes2, votingContract)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Time to create snapshot: %v\n", time.Since(start))

	store.Close()
	snapshotStore.Close()
	chainStore.Close()

	// copy other state data (not pruned)
	fmt.Println("Copying the rest of the chain data (chain, statesql)...")
	copyDir(chainPath, snapshotChainPath)
	copyDir(sqlPath, snapshotSqlPath)

	// display results of general trie info
	displayResults(sa, contractTrie)
	displayFolderSizes(dbPath, "Size information BEFORE snapshot:")
	displayFolderSizes(snapshotPath, "Size information AFTER snapshot:")

	/* Sample code to use when pruning chain db and resetting latest

	// Prune chain data
	blockIdx := types.BlockNoToBytes(11758998)
	block0Idx := types.BlockNoToBytes(0)

	// set snapshot chain block
	chainDbPath := path.Join(dbPath, "chain")
	chainStore := db.NewDB(db.BadgerImpl, chainDbPath)
	blockHash := chainStore.Get(blockIdx)
	block0Hash := chainStore.Get(block0Idx)
	if len(blockHash) == 0 {
		fmt.Println("block hash")
		return
	}
	//block := types.Block{}
	raw := chainStore.Get(blockHash)
	raw0 := chainStore.Get(block0Hash)
	if raw == nil || len(raw) == 0 {
		fmt.Println("failed to load block data")
		return
	}
		err = proto.Unmarshal(raw, block)
		if err != nil {
			fmt.Println("failed to unmarshall block")
			return
		}
		if !bytes.Equal(block.Hash, blockHash) {
			fmt.Println("loaded block doest't have expected hash")
			return
		}
	genesisKey := []byte("chain.genesisInfo")
	genesisBalanceKey := []byte("chain.genesisBalance")
	genesis := chainStore.Get(genesisKey)
	genesisBalance := chainStore.Get(genesisBalanceKey)
	chainStore.Close()

	snapshotChainDbPath := path.Join(snapshotPath, "chain")
	err = os.MkdirAll(snapshotChainDbPath, 0755)
	if err != nil {
		fmt.Println("Enable to create snapshot state folder")
		return
	}
	snapshotChainStore := db.NewDB(db.BadgerImpl, snapshotChainDbPath)
	// set latest
	latestKey := []byte("chain.latest")
	snapshotChainStore.Set(latestKey, blockIdx)
	snapshotChainStore.Set(blockIdx, blockHash)
	snapshotChainStore.Set(blockHash, raw)
	snapshotChainStore.Set(block0Idx, block0Hash)
	snapshotChainStore.Set(block0Hash, raw0)
	snapshotChainStore.Set(genesisKey, genesis)
	snapshotChainStore.Set(genesisBalanceKey, genesisBalance)
	snapshotChainStore.Close()
	*/
}
