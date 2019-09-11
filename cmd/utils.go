package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/aergoio/aergo-lib/db"
	"github.com/aergoio/aergo/types"
	"github.com/aergoio/state-tools/stool"
	"github.com/gogo/protobuf/proto"
)

func isEmpty(name string) bool {
	f, err := os.Open(name)
	defer f.Close()
	if err != nil {
		return false
	}
	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true
	}
	return false
}

func dirSize(path string) (int64, error) {
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

func displayResults(sa *stool.StateAnalysis, contractTrie bool) {
	if contractTrie {
		fmt.Println("\nContract trie analysis results:")
		fmt.Println("=================================")
		fmt.Println("* Number of keys in the trie: ", sa.Counters.NbStorageValues)
		if counterOn {
			fmt.Println("* Number of DB reads performed to iterate Trie: ", sa.Trie.LoadDbCounter)
		}

	} else {
		fmt.Println("\nGeneral trie analysis results:")
		fmt.Println("==============================")
		fmt.Println("* Number of contracts: ", sa.Counters.NbContracts)
		fmt.Println("* Number of pubKey accounts + 1 (staking contract): ", sa.Counters.NbUserAccounts)
		fmt.Println("* Number of 0 balance pubkeys: ", sa.Counters.NbUserAccounts0)
		fmt.Println("* Total number of accounts (pubkey + contract): ", sa.Counters.NbUserAccounts0+sa.Counters.NbUserAccounts+sa.Counters.NbContracts)
		fmt.Println("* Number of nil (0 nonce, 0 balance) objects: ", sa.Counters.NbNilObjects)
		if counterOn {
			fmt.Println("* Number of DB reads performed to iterate Trie: ", sa.Trie.LoadDbCounter)
		}
		fmt.Println("* Total Aer Balance of all pubKeys and contracts: ", sa.Counters.TotalAerBalance)
	}
}

func displayFolderSizes(dbPath, title string) {
	statePath := path.Join(dbPath, "state")
	sqlPath := path.Join(dbPath, "statesql")
	chainPath := path.Join(dbPath, "chain")
	totalSize, _ := dirSize(dbPath)
	stateSize, _ := dirSize(statePath)
	sqlSize, _ := dirSize(sqlPath)
	chainSize, _ := dirSize(chainPath)
	fmt.Printf("\n%s\n", title)
	fmt.Println(strings.Repeat("=", len(title)))
	fmt.Println("* Total blockchain size: ", float64(totalSize)/1024.0/1024.0, " Mb")
	fmt.Println("* State size: ", float64(stateSize)/1024.0/1024.0, " Mb")
	fmt.Println("* Chain size: ", float64(chainSize)/1024.0/1024.0, " Mb")
	fmt.Println("* SQL State size: ", float64(sqlSize)/1024.0/1024.0, " Mb")
}

func copyDir(sourcePath, destinationPath string) {
	exec.Command("cp", "-r", sourcePath, destinationPath).Run()
}

func getLatestTrieRoot(chainPath string) ([]byte, error) {
	chainStore := db.NewDB(db.BadgerImpl, chainPath)
	latestKey := []byte("chain.latest")
	blockIdx := chainStore.Get(latestKey)
	if blockIdx == nil || len(blockIdx) == 0 {
		return nil, fmt.Errorf("failed to load latest blockidx")
	}

	//blockNo := types.BlockNoFromBytes(blockIdx)
	blockHash := chainStore.Get(blockIdx)
	blockRaw := chainStore.Get(blockHash)
	if blockRaw == nil || len(blockRaw) == 0 {
		return nil, fmt.Errorf("failed to load latest block data")
	}
	block := types.Block{}
	err := proto.Unmarshal(blockRaw, &block)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall block")
	}
	if !bytes.Equal(block.Hash, blockHash) {
		return nil, fmt.Errorf("loaded block doest't have expected hash")
	}
	return block.Header.BlocksRootHash, nil
}
