# state-tools

This tool iterates all the leaves in a state Trie stored in the aergo state db.
It can simply analyse leaves in a state root (general or contract trie) or create a snapshot of the whole database while keeping only the 'latest' state.


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
```

### State analysis
#### Default: analyse the latest General trie state
```sh
$ state-tools analysis -p .aergo/data

Analysing state with root:  9u4XgnVxFw4nmeXqYbs5HGNHGg7YPfgK5JgrVLX2Nrc7

General trie analysis results:
==============================
* Number of contracts:  14339
* Number of pubKey accounts + 1 (staking contract):  4327
* Number of 0 balance pubkeys:  85
* Total number of accounts (pubkey + contract):  18751
* Number of nil (0 nonce, 0 balance) objects:  5
* Total Aer Balance of all pubKeys and contracts:  500000000000000000000000000
* Average trie depth: 15.49
* Deepest leaf in the trie:  32
* Number of DB reads performed to iterate Trie:  10459

Current latest state size information:
======================================
* Total blockchain size:  9165.879945755005  Mb
* State size:  431.803409576416  Mb
* Chain size:  8045.8540391922  Mb
* SQL State size:  688.21875  Mb
```

#### Analyse the leaves of a given root (general or contract trie)
```sh
$ state-tools analysis -p .aergo/data -r 9u4XgnVxFw4nmeXqYbs5HGNHGg7YPfgK5JgrVLX2Nrc7
```

#### Analyse the general trie state at a given block height
```sh
$ state-tools analysis -p .aergo/data -b 2222
```


### State snapshot
Currently only state trie data is pruned, chain data and sql data are simply copied
```sh
$ state-tools snapshot -p .aergo/data -s snapshot/.aergo/data

Iterating the Aergo state trie to create snapshot...
Time to create snapshot: 9.477358269s
Copying the rest of the chain data (chain, statesql)...

General trie analysis results:
==============================
* Number of contracts:  14276
* Number of pubKey accounts + 1 (staking contract):  4327
* Number of 0 balance pubkeys:  85
* Total number of accounts (pubkey + contract):  18688
* Number of nil (0 nonce, 0 balance) objects:  5
* Total Aer Balance of all pubKeys and contracts:  500000000000000000000000000
* Average trie depth: 15.48
* Deepest leaf in the trie:  32
* Number of DB reads performed to iterate Trie:  10428

Size information BEFORE snapshot:
=================================
* Total blockchain size:  8921.005863189697  Mb
* State size:  397.4595766067505  Mb
* Chain size:  7837.4362535476685  Mb
* SQL State size:  686.109375  Mb

Size information AFTER snapshot:
================================
* Total blockchain size:  8560.56925868988  Mb
* State size:  37.023630142211914  Mb
* Chain size:  7837.4362535476685  Mb
* SQL State size:  686.109375  Mb
```