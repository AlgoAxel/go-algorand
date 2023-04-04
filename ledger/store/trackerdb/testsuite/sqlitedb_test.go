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

package testsuite

import (
	"context"
	"fmt"
	"testing"

	"github.com/algorand/go-algorand/data/basics"
	"github.com/algorand/go-algorand/ledger/store/trackerdb"
	"github.com/algorand/go-algorand/ledger/store/trackerdb/sqlitedriver"
	"github.com/algorand/go-algorand/protocol"
	"github.com/stretchr/testify/require"
)

func TestSqliteDB(t *testing.T) {
	dbFactory := func() dbForTests {
		// create a tmp dir for the db, the testing runtime will clean it up automatically
		fn := fmt.Sprintf("%s/tracker-db.sqlite", t.TempDir())
		db, err := sqlitedriver.OpenTrackerSQLStore(fn, false)
		require.NoError(t, err)

		// initialize db
		err = db.Transaction(func(ctx context.Context, tx trackerdb.TransactionScope) (err error) {
			accounts := make(map[basics.Address]basics.AccountData)
			tx.Testing().AccountsInitTest(t, accounts, protocol.ConsensusCurrentVersion)

			return nil
		})
		require.NoError(t, err)

		// TODO: we should eventually move hte sql to use the seed_db()

		return db
	}

	// run the suite
	runGenericTestsWithDB(t, dbFactory)
}