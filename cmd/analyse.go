package cmd

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/aergoio/aergo-lib/db"
	"github.com/aergoio/state-tools/stool"
	"github.com/mr-tron/base58/base58"
	"github.com/spf13/cobra"
)

var (
	contractTrie bool
)

func init() {
	analyseCmd.Flags().BoolVar(&contractTrie, "contractTrie", false, "The trie being queried is the general trie")
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

	if contractTrie {
		fmt.Println("\nContract trie analysis results:")
		fmt.Println("=================================")
		fmt.Println("Number of keys in the trie: ", sa.Counters.NbStorageValues)
		if counterOn {
			fmt.Println("Number of DB reads performed to iterate Trie: ", sa.Trie.LoadDbCounter)
		}

	} else {
		fmt.Println("\nGeneral trie analysis results:")
		fmt.Println("==============================")
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

	sqlPath := path.Join(dbPath, "statesql")
	chainPath := path.Join(dbPath, "chain")
	accPath := path.Join(dbPath, "account")
	totalSize, err := DirSize(dbPath)
	stateSize, err := DirSize(statePath)
	sqlSize, err := DirSize(sqlPath)
	chainSize, err := DirSize(chainPath)
	accSize, err := DirSize(accPath)
	fmt.Println("\nOther information:")
	fmt.Println("==================")
	fmt.Println("Total blockchain size: ", float64(totalSize)/1024.0/1024.0, " Mb")
	fmt.Println("State size: ", float64(stateSize)/1024.0/1024.0, " Mb")
	fmt.Println("Chain size: ", float64(chainSize)/1024.0/1024.0, " Mb")
	fmt.Println("SQL State size: ", float64(sqlSize)/1024.0/1024.0, " Mb")
	fmt.Println("Account size: ", float64(accSize)/1024.0/1024.0, " Mb")
}

func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}
