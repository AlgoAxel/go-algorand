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

package transactions

// MessUpSigForTesting will mess up the signature so that Verify() will fail on the signed transaction.
// Intended to be used in tests outside the transactions package (e.g., block validation tests) where we want to check whether we're verifying signatures.
func (stxn *SignedTxn) MessUpSigForTesting() {
	stxn.Sig[0] ^= 1
}
