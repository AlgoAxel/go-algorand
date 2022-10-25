// Copyright (C) 2019-2022 Algorand, Inc.
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

package network

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"github.com/algorand/go-algorand/crypto"
	"github.com/algorand/go-algorand/protocol"
)

// Initial Deduplication Header
const ProtocolConectionIdentityChallengeHeader = "X-Algorand-IdentityChallenge"

type identityChallenge struct {
	Nonce     int              `codec:"n"`
	Key       crypto.PublicKey `codec:"pk"`
	Challenge [32]byte         `codec:"c"`
	Signature crypto.Signature `coded:"s"`
}

type identityChallengeResponse struct {
	identityChallenge
	ResponseChallenge [32]byte `codec:"rc"`
}

func NewIdentityChallenge(p crypto.PublicKey) identityChallenge {
	c := identityChallenge{
		Nonce:     1,
		Key:       p,
		Challenge: [32]byte{},
	}
	crypto.RandBytes(c.Challenge[:])
	return c
}

func (i identityChallenge) signableBytes() []byte {
	return bytes.Join([][]byte{
		[]byte(fmt.Sprintf("%d", i.Nonce)),
		i.Challenge[:],
		i.Key[:],
	},
		[]byte(":"))
}

func (i identityChallenge) sign(s *crypto.SignatureSecrets) crypto.Signature {
	return s.SignBytes(i.signableBytes())
}

func (i identityChallenge) verify() error {
	b := i.signableBytes()
	verified := i.Key.VerifyBytes(b, i.Signature)
	if !verified {
		return fmt.Errorf("included signature does not verify identity challenge")
	}
	return nil
}

// SignAndEncodeB64 signs the identityChallenge, attaches a signature, and converts
// the structure to a b64 and msgpk'd string to be included as a header
func (i *identityChallenge) SignAndEncodeB64(s *crypto.SignatureSecrets) string {
	i.Signature = i.sign(s)
	enc := protocol.EncodeReflect(i)
	b64enc := base64.StdEncoding.EncodeToString(enc)
	return b64enc
}

// IdentityChallengeFromB64 will return an Identity Challenge from the B64 header string
func IdentityChallengeFromB64(i string) identityChallenge {
	msg, err := base64.StdEncoding.DecodeString(i)
	if err != nil {
		return identityChallenge{}
	}
	ret := identityChallenge{}
	protocol.DecodeReflect(msg, &ret)
	return ret
}

func NewIdentityChallengeResponse(p crypto.PublicKey, id identityChallenge) identityChallengeResponse {
	c := identityChallengeResponse{
		identityChallenge: identityChallenge{
			Nonce:     2,
			Key:       p,
			Challenge: id.Challenge,
		},
		ResponseChallenge: [32]byte{},
	}
	crypto.RandBytes(c.ResponseChallenge[:])
	return c
}

func (i identityChallengeResponse) signableBytes() []byte {
	return bytes.Join([][]byte{
		[]byte(fmt.Sprintf("%d", i.Nonce)),
		i.Challenge[:],
		i.ResponseChallenge[:],
		i.Key[:],
	},
		[]byte(":"))
}

func (i identityChallengeResponse) sign(s *crypto.SignatureSecrets) crypto.Signature {
	return s.SignBytes(i.signableBytes())
}

func (i identityChallengeResponse) verify() error {
	b := i.signableBytes()
	verified := i.Key.VerifyBytes(b, i.Signature)
	if !verified {
		return fmt.Errorf("included signature does not verify identity challenge")
	}
	return nil
}

func IdentityChallengeResponseFromB64(i string) identityChallengeResponse {
	msg, err := base64.StdEncoding.DecodeString(i)
	if err != nil {
		return identityChallengeResponse{}
	}
	ret := identityChallengeResponse{}
	protocol.DecodeReflect(msg, &ret)
	return ret
}

func (i *identityChallengeResponse) SignAndEncodeB64(s *crypto.SignatureSecrets) string {
	i.Signature = i.sign(s)
	enc := protocol.EncodeReflect(i)
	b64enc := base64.StdEncoding.EncodeToString(enc)
	return b64enc
}

func CheckPeerValidation() {
	return
}
