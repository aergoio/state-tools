package cmd

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/aergoio/aergo-lib/db"
	"github.com/aergoio/state-tools/stool"
	"github.com/mr-tron/base58/base58"
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
	rootBytes, _ := base58.Decode(root)

	// check db path
	statePath := path.Join(dbPath, "state")
	if stat, err := os.Stat(statePath); err != nil || !stat.IsDir() {
		fmt.Println("Invalid database path provided")
		return
	}
	// check snapshot path
	if stat, err := os.Stat(snapshotPath); err != nil || !stat.IsDir() {
		fmt.Println("Invalid path for snapshot database provided")
		return
	}
	if !isEmpty(snapshotPath) {
		fmt.Println("Snapshot folder must be empty")
		return
	}
	snapshotStatePath := path.Join(snapshotPath, "state")
	err := os.MkdirAll(snapshotStatePath, 0755)
	if err != nil {
		fmt.Println("Enable to create snapshot state folder")
		return
	}

	store := db.NewDB(db.BadgerImpl, statePath)
	snapStore := db.NewDB(db.BadgerImpl, snapshotStatePath)
	sa := stool.NewStateAnalysis(store, counterOn, !contractTrie, 10000)

	fmt.Println("Iterating the Aergo state trie to create snapshot...")
	start := time.Now()
	err = sa.Snapshot(snapStore, rootBytes)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Time to create snapshot: %v\n", time.Since(start))

	store.Close()
	snapStore.Close()

	// copy other state data (not pruned)
	fmt.Println("Copying the rest of the chain data (account, chain, statesql)...")
	copyDir(path.Join(dbPath, "account"), path.Join(snapshotPath, "account"))
	copyDir(path.Join(dbPath, "chain"), path.Join(snapshotPath, "chain"))
	copyDir(path.Join(dbPath, "statesql"), path.Join(snapshotPath, "statesql"))

	// display results of general trie info
	displayResults(sa, contractTrie)
	displayFolderSizes(dbPath, "Size information BEFORE snapshot:")
	displayFolderSizes(snapshotPath, "Size information AFTER snapshot:")
}
