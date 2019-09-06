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
	counterOn := false
	snapshot := false

	fmt.Printf("\nPath/to/aergo/database/data/state:\n> ")
	fmt.Scanln(&dbPath)
	fmt.Printf("\nBlock's TrieRoot (b58):\n> ")
	fmt.Scanln(&stateRoot)
	fmt.Printf("\nCount DB reads (y/n):\n> ")
	fmt.Scanln(&countDbReads)
	if countDbReads == "y" || countDbReads == "yes" || countDbReads == "Y" || countDbReads == "Yes" {
		counterOn = true
	}

	fmt.Printf("\nAlso snapshot the state? This will copy all the data recorded under TrieRoot (y/n):\n> ")
	fmt.Scanln(&createSnapshot)
	if createSnapshot == "y" || createSnapshot == "yes" || createSnapshot == "Y" || createSnapshot == "Yes" {
		fmt.Printf("\nSnapshot/path/to/aergo/database/data/state:\n> ")
		fmt.Scanln(&dbSnapshotPath)
		snapshot = true
	}

	// dbPath := "/Users/pa/Go_workspace/src/github.com/aergoio/aergo/.aergo/data/state"
	// stateRoot := "66QMsEUpLc7zx2D4jkYMN5UbGefmdFFHePnXrZitKJc2"

	rootBytes, _ := base58.Decode(stateRoot)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		_ = os.MkdirAll(dbPath, 0711)
	}
	store := db.NewDB(db.BadgerImpl, dbPath)
	aa := stool.NewStateAnalysis(store, counterOn, snapshot)
	err := aa.Dfs(rootBytes, 0, 256, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("\nAnalysis results:")
	fmt.Println("=================")
	fmt.Println("Number of contracts: ", aa.NbContracts)
	fmt.Println("Number of pubKey accounts + 1 (staking contract): ", aa.NbUserAccounts)
	fmt.Println("Number of 0 balance pubkeys: ", aa.NbUserAccounts0)
	fmt.Println("Total number of accounts (pubkey + contract): ", aa.NbUserAccounts0+aa.NbUserAccounts+aa.NbContracts)
	fmt.Println("Number of nil (0 nonce, 0 balance) objects: ", aa.NbNilObjects)
	if counterOn {
		fmt.Println("Number of DB reads performed to iterate Trie: ", aa.Trie.LoadDbCounter)
	}
	fmt.Println("Total Aer Balance of all pubKeys and contracts: ", aa.TotalAerBalance)

	store.Close()
}
