package main

import (
	"fmt"
	"os"

	"github.com/aergoio/aergo-lib/db"
	"github.com/aergoio/state-tools/analysis"
	"github.com/mr-tron/base58/base58"
)

func main() {
	fmt.Println("Welcome to the Aergo state analysis tool:")
	fmt.Println("=========================================")
	fmt.Println("=========================================")

	var dbPath string
	var stateRoot string
	var countDbReads string
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
	// dbPath := "/Users/pa/Go_workspace/src/github.com/aergoio/aergo/.aergo/data/state"
	// stateRoot := "66QMsEUpLc7zx2D4jkYMN5UbGefmdFFHePnXrZitKJc2"

	rootBytes, _ := base58.Decode(stateRoot)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		_ = os.MkdirAll(dbPath, 0711)
	}
	store := db.NewDB(db.BadgerImpl, dbPath)
	aa := analysis.NewAccountsAnalysis(store, counterOn)
	err := aa.Dfs(rootBytes, 0, 256, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("\nAnalysis results:")
	fmt.Println("=================")
	fmt.Println("Number of contracts: ", aa.NbContracts)
	fmt.Println("Number of pubKey accounts: ", aa.NbUserAccounts)
	fmt.Println("Total number of accounts (pubkey + contract): ", aa.NbUserAccounts+aa.NbContracts)
	fmt.Println("Number of other objects stored in trie: ", aa.NbOtherObjects)
	if counterOn {
		fmt.Println("Number of DB reads performed to iterate Trie: ", aa.Trie.LoadDbCounter)
	}
	fmt.Println("Total Aer Balance of all pubKeys and contracts: ", aa.TotalAerBalance)

	store.Close()
}
