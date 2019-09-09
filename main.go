package main

import (
	"fmt"
	"os"

	"github.com/aergoio/aergo-lib/db"
	"github.com/aergoio/state-tools/stool"
	"github.com/mr-tron/base58/base58"
)

func main() {
	fmt.Println("Welcome to the Aergo snapshot and state analysis tool:")
	fmt.Println("======================================================")
	fmt.Println("======================================================")

	var dbPath string
	var stateRoot string
	var countDbReads string
	var createSnapshot string
	var dbSnapshotPath string
	var snapStore db.DB
	counterOn := false

	fmt.Printf("\nPath/to/aergo/database/data/state:\n> ")
	fmt.Scanln(&dbPath)
	fmt.Printf("\nBlock's TrieRoot (b58):\n> ")
	fmt.Scanln(&stateRoot)
	fmt.Printf("\nCount DB reads (y/n):\n> ")
	fmt.Scanln(&countDbReads)
	if countDbReads == "y" || countDbReads == "yes" || countDbReads == "Y" || countDbReads == "Yes" {
		counterOn = true
	}
	rootBytes, _ := base58.Decode(stateRoot)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		_ = os.MkdirAll(dbPath, 0711)
	}
	store := db.NewDB(db.BadgerImpl, dbPath)
	sa := stool.NewStateAnalysis(store, counterOn, true, 10000)

	fmt.Printf("\nAlso snapshot the state? This will copy all the data recorded under TrieRoot (y/n):\n> ")
	fmt.Scanln(&createSnapshot)
	if createSnapshot == "y" || createSnapshot == "yes" || createSnapshot == "Y" || createSnapshot == "Yes" {
		fmt.Printf("\nSnapshot/path/to/aergo/database/data/state:\n> ")
		fmt.Scanln(&dbSnapshotPath)

		// create snapshot
		snapStore = db.NewDB(db.BadgerImpl, dbSnapshotPath)
		err := sa.Snapshot(snapStore, rootBytes)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {

		// Just iterate storage
		err := sa.Analyse(rootBytes)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	fmt.Println("\nGeneral trie analysis results:")
	fmt.Println("=================")
	fmt.Println("Number of contracts: ", sa.Counters.NbContracts)
	fmt.Println("Number of pubKey accounts + 1 (staking contract): ", sa.Counters.NbUserAccounts)
	fmt.Println("Number of 0 balance pubkeys: ", sa.Counters.NbUserAccounts0)
	fmt.Println("Total number of accounts (pubkey + contract): ", sa.Counters.NbUserAccounts0+sa.Counters.NbUserAccounts+sa.Counters.NbContracts)
	fmt.Println("Number of nil (0 nonce, 0 balance) objects: ", sa.Counters.NbNilObjects)
	if counterOn {
		fmt.Println("Number of DB reads performed to iterate Trie: ", sa.Trie.LoadDbCounter)
	}
	fmt.Println("Total Aer Balance of all pubKeys and contracts: ", sa.Counters.TotalAerBalance)

	store.Close()
}
