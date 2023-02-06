// Copyright (C) 2019-2023 Algorand, Inc.
// This file is part of go-algorand
//
// go-algorand is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// go-algorand is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with go-algorand.  If not, see <https://www.gnu.org/licenses/>.

package store

import (
	"context"
	"testing"

	"github.com/algorand/go-algorand/config"
	"github.com/algorand/go-algorand/crypto"
	"github.com/algorand/go-algorand/data/basics"
	"github.com/algorand/go-algorand/ledger/ledgercore"
)

// AccountsWriter is the write interface for:
// - accounts, resources, app kvs, creatables
type AccountsWriter interface {
	InsertAccount(addr basics.Address, normBalance uint64, data BaseAccountData) (rowid int64, err error)
	DeleteAccount(rowid int64) (rowsAffected int64, err error)
	UpdateAccount(rowid int64, normBalance uint64, data BaseAccountData) (rowsAffected int64, err error)

	InsertResource(addrid int64, aidx basics.CreatableIndex, data ResourcesData) (rowid int64, err error)
	DeleteResource(addrid int64, aidx basics.CreatableIndex) (rowsAffected int64, err error)
	UpdateResource(addrid int64, aidx basics.CreatableIndex, data ResourcesData) (rowsAffected int64, err error)

	UpsertKvPair(key string, value []byte) error
	DeleteKvPair(key string) error

	InsertCreatable(cidx basics.CreatableIndex, ctype basics.CreatableType, creator []byte) (rowid int64, err error)
	DeleteCreatable(cidx basics.CreatableIndex, ctype basics.CreatableType) (rowsAffected int64, err error)

	Close()
}

// AccountsWriterExt is the write interface used inside transactions and batch operations.
type AccountsWriterExt interface {
	AccountsReset(ctx context.Context) error
	ResetAccountHashes(ctx context.Context) (err error)
	TxtailNewRound(ctx context.Context, baseRound basics.Round, roundData [][]byte, forgetBeforeRound basics.Round) error
	UpdateAccountsRound(rnd basics.Round) (err error)
	UpdateAccountsHashRound(ctx context.Context, hashRound basics.Round) (err error)
	AccountsPutTotals(totals ledgercore.AccountTotals, catchpointStaging bool) error
	OnlineAccountsDelete(forgetBefore basics.Round) (err error)
	AccountsPutOnlineRoundParams(onlineRoundParamsData []ledgercore.OnlineRoundParamsData, startRound basics.Round) error
	AccountsPruneOnlineRoundParams(deleteBeforeRound basics.Round) error
}

// AccountsReader is the read interface for:
// - accounts, resources, app kvs, creatables
type AccountsReader interface {
	ListCreatables(maxIdx basics.CreatableIndex, maxResults uint64, ctype basics.CreatableType) (results []basics.CreatableLocator, dbRound basics.Round, err error)

	LookupAccount(addr basics.Address) (data PersistedAccountData, err error)

	LookupResources(addr basics.Address, aidx basics.CreatableIndex, ctype basics.CreatableType) (data PersistedResourcesData, err error)
	LookupAllResources(addr basics.Address) (data []PersistedResourcesData, rnd basics.Round, err error)

	LookupKeyValue(key string) (pv PersistedKVData, err error)
	LookupKeysByPrefix(prefix string, maxKeyNum uint64, results map[string]bool, resultCount uint64) (round basics.Round, err error)

	LookupCreator(cidx basics.CreatableIndex, ctype basics.CreatableType) (addr basics.Address, ok bool, dbRound basics.Round, err error)

	Close()
}

type AccountsReaderExt interface {
	AccountsTotals(ctx context.Context, catchpointStaging bool) (totals ledgercore.AccountTotals, err error)
	AccountsHashRound(ctx context.Context) (hashrnd basics.Round, err error)
	LookupAccountAddressFromAddressID(ctx context.Context, addrid int64) (address basics.Address, err error)
	LookupAccountDataByAddress(basics.Address) (rowid int64, data []byte, err error)
	LookupAccountRowID(basics.Address) (addrid int64, err error)
	LookupResourceDataByAddrID(addrid int64, aidx basics.CreatableIndex) (data []byte, err error)
	TotalAccounts(ctx context.Context) (total uint64, err error)
	TotalKVs(ctx context.Context) (total uint64, err error)
	AccountsRound() (rnd basics.Round, err error)
	AccountsAllTest() (bals map[basics.Address]basics.AccountData, err error)
	CheckCreatablesTest(t *testing.T, iteration int, expectedDbImage map[basics.CreatableIndex]ledgercore.ModifiedCreatable)
	LookupOnlineAccountDataByAddress(addr basics.Address) (rowid int64, data []byte, err error)
	AccountsOnlineTop(rnd basics.Round, offset uint64, n uint64, proto config.ConsensusParams) (map[basics.Address]*ledgercore.OnlineAccount, error)
	AccountsOnlineRoundParams() (onlineRoundParamsData []ledgercore.OnlineRoundParamsData, endRound basics.Round, err error)
	OnlineAccountsAll(maxAccounts uint64) ([]PersistedOnlineAccountData, error)
	LoadTxTail(ctx context.Context, dbRound basics.Round) (roundData []*TxTailRound, roundHash []crypto.Digest, baseRound basics.Round, err error)
	LoadAllFullAccounts(ctx context.Context, balancesTable string, resourcesTable string, acctCb func(basics.Address, basics.AccountData)) (count int, err error)
}

// AccountsReaderWriter is AccountsReader+AccountsWriter
type AccountsReaderWriter interface {
	// AccountsReader
	// AccountsWriter
	AccountsWriterExt
	AccountsReaderExt
}

// OnlineAccountsWriter is the write interface for:
// - online accounts
type OnlineAccountsWriter interface {
	InsertOnlineAccount(addr basics.Address, normBalance uint64, data BaseOnlineAccountData, updRound uint64, voteLastValid uint64) (rowid int64, err error)

	Close()
}

// OnlineAccountsReader is the read interface for:
// - online accounts
type OnlineAccountsReader interface {
	LookupOnline(addr basics.Address, rnd basics.Round) (data PersistedOnlineAccountData, err error)
	LookupOnlineTotalsHistory(round basics.Round) (basics.MicroAlgos, error)
	LookupOnlineHistory(addr basics.Address) (result []PersistedOnlineAccountData, rnd basics.Round, err error)

	Close()
}

// CatchpointWriter is the write interface for:
// - catchpoints
type CatchpointWriter interface {
	CreateCatchpointStagingHashesIndex(ctx context.Context) (err error)

	StoreCatchpoint(ctx context.Context, round basics.Round, fileName string, catchpoint string, fileSize int64) (err error)

	WriteCatchpointStateUint64(ctx context.Context, stateName CatchpointState, setValue uint64) (err error)
	WriteCatchpointStateString(ctx context.Context, stateName CatchpointState, setValue string) (err error)

	WriteCatchpointStagingBalances(ctx context.Context, bals []NormalizedAccountBalance) error
	WriteCatchpointStagingKVs(ctx context.Context, keys [][]byte, values [][]byte, hashes [][]byte) error
	WriteCatchpointStagingCreatable(ctx context.Context, bals []NormalizedAccountBalance) error
	WriteCatchpointStagingHashes(ctx context.Context, bals []NormalizedAccountBalance) error

	ApplyCatchpointStagingBalances(ctx context.Context, balancesRound basics.Round, merkleRootRound basics.Round) (err error)
	ResetCatchpointStagingBalances(ctx context.Context, newCatchup bool) (err error)

	InsertUnfinishedCatchpoint(ctx context.Context, round basics.Round, blockHash crypto.Digest) error
	DeleteUnfinishedCatchpoint(ctx context.Context, round basics.Round) error
	DeleteOldCatchpointFirstStageInfo(ctx context.Context, maxRoundToDelete basics.Round) error
	InsertOrReplaceCatchpointFirstStageInfo(ctx context.Context, round basics.Round, info *CatchpointFirstStageInfo) error

	DeleteStoredCatchpoints(ctx context.Context, dbDirectory string) (err error)
}

// CatchpointReader is the read interface for:
// - catchpoints
type CatchpointReader interface {
	GetCatchpoint(ctx context.Context, round basics.Round) (fileName string, catchpoint string, fileSize int64, err error)
	GetOldestCatchpointFiles(ctx context.Context, fileCount int, filesToKeep int) (fileNames map[basics.Round]string, err error)

	ReadCatchpointStateUint64(ctx context.Context, stateName CatchpointState) (val uint64, err error)
	ReadCatchpointStateString(ctx context.Context, stateName CatchpointState) (val string, err error)

	SelectUnfinishedCatchpoints(ctx context.Context) ([]UnfinishedCatchpointRecord, error)
	SelectCatchpointFirstStageInfo(ctx context.Context, round basics.Round) (CatchpointFirstStageInfo, bool /*exists*/, error)
	SelectOldCatchpointFirstStageInfoRounds(ctx context.Context, maxRound basics.Round) ([]basics.Round, error)
}

// CatchpointReaderWriter is CatchpointReader+CatchpointWriter
type CatchpointReaderWriter interface {
	CatchpointReader
	CatchpointWriter
}
