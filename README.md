# state-tools

This tool iterates all the leaves in a state Trie (of given root) stored in the aergo state db.


## Installation


```sh
$ cd $GOPATH/src/github.com/aergoio/state-tools
$ go install
```

## Usage

```sh
$ state-tools -h       

state-tools analyses and creates snapshots of aergo state at a given trie root
                   When parsing the general trie, accounts are analysed to count all account types and balances.
                   Snapshots can be created for the whole state and stored in a new database.
                   Other statistics about blockchain state and chain data size are also provided.

Usage:
  state-tools [command]

Available Commands:
  analyse     Analyse the database
  help        Help about any command
  snapshot    Create a snapshot of the database
  version     Print the version number of state-tools

Flags:
  -c, --counterOn       Make a counter of db reads (default true)
  -p, --dbPath string   Path/to/blockchain/database/folder/data
  -h, --help            help for state-tools

Use "state-tools [command] --help" for more information about a command.


$ state-tools analysis -p .aergo/data

General trie analysis results:
==============================
* Number of contracts:  14341
* Number of pubKey accounts + 1 (staking contract):  4325
* Number of 0 balance pubkeys:  85
* Total number of accounts (pubkey + contract):  18751
* Number of nil (0 nonce, 0 balance) objects:  5
* Number of DB reads performed to iterate Trie:  10459
* Total Aer Balance of all pubKeys and contracts:  500000000000000000000000000

Size information:
=================
* Total blockchain size:  8727.776446342468  Mb
* State size:  59.62231636047363  Mb
* Chain size:  7979.934971809387  Mb
* SQL State size:  688.21875  Mb

```