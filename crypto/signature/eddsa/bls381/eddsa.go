// Copyright 2020 ConsenSys Software Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eddsa

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash"
	"math/big"

	"github.com/consensys/gurvy/bls381/twistededwards"
	"golang.org/x/crypto/blake2b"
)

var errNotOnCurve = errors.New("point not on curve")

const frSize = 32 // TODO assumes a 256 bits field for the twisted curve (ok for our implem)

// Signature represents an eddsa signature
// cf https://en.wikipedia.org/wiki/EdDSA for notation
type Signature struct {
	R twistededwards.Point
	S big.Int
}

// PublicKey eddsa signature object
// cf https://en.wikipedia.org/wiki/EdDSA for notation
type PublicKey struct {
	A     twistededwards.Point
	HFunc hash.Hash
}

// PrivateKey private key of an eddsa instance
type PrivateKey struct {
	randSrc [32]byte // randomizer (non need to convert it when doing scalar mul --> random = H(randSrc,msg))
	scalar  big.Int  // secret scalar (non need to convert it when doing scalar mul)
}

// GetCurveParams get the parameters of the Edwards curve used
func GetCurveParams() twistededwards.CurveParams {
	return twistededwards.GetEdwardsCurve()
}

// New creates an instance of eddsa
func New(seed [32]byte, hFunc hash.Hash) (PublicKey, PrivateKey) {

	c := GetCurveParams()

	var pub PublicKey
	var priv PrivateKey

	h := blake2b.Sum512(seed[:])
	for i := 0; i < 32; i++ {
		priv.randSrc[i] = h[i+32]
	}

	// prune the key
	// https://tools.ietf.org/html/rfc8032#section-5.1.5, key generation
	h[0] &= 0xF8
	h[31] &= 0x7F
	h[31] |= 0x40

	// reverse first bytes because setBytes interpret stream as big endian
	// but in eddsa specs s is the first 32 bytes in little endian
	for i, j := 0, 32; i < j; i, j = i+1, j-1 {
		h[i], h[j] = h[j], h[i]
	}
	priv.scalar.SetBytes(h[:32])

	pub.A.ScalarMul(&c.Base, &priv.scalar)
	pub.HFunc = hFunc

	return pub, priv
}

// Sign sign a message
// cf https://en.wikipedia.org/wiki/EdDSA for the notations
// Eddsa is supposed to be built upon Edwards (or twisted Edwards) curves having 256 bits group size and cofactor=4 or 8
func Sign(message []byte, pub PublicKey, priv PrivateKey) (Signature, error) {

	curveParams := GetCurveParams()

	res := Signature{}

	var randScalarInt big.Int

	// randSrc = privKey.randSrc || msg (-> message = MSB message .. LSB message)
	randSrc := make([]byte, 64)
	for i, v := range priv.randSrc {
		randSrc[i] = v
	}
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, message)
	if err != nil {
		return res, err
	}
	bufb := buf.Bytes()
	for i := 0; i < 32; i++ {
		randSrc[32+i] = bufb[i]
	}

	// randBytes = H(randSrc)
	randBytes := blake2b.Sum512(randSrc[:]) // TODO ensures that the hash used to build the key and the one used here is the same
	randScalarInt.SetBytes(randBytes[:32])

	// compute R = randScalar*Base
	res.R.ScalarMul(&curveParams.Base, &randScalarInt)
	if !res.R.IsOnCurve() {
		return Signature{}, errNotOnCurve
	}

	// compute H(R, A, M), all parameters in data are in Montgomery form
	resRX := res.R.X.Bytes()
	resRY := res.R.Y.Bytes()
	resAX := pub.A.X.Bytes()
	resAY := pub.A.Y.Bytes()
	sizeDataToHash := 4*frSize + len(message)
	dataToHash := make([]byte, sizeDataToHash)
	copy(dataToHash[:], resRX[:])
	copy(dataToHash[frSize:], resRY[:])
	copy(dataToHash[2*frSize:], resAX[:])
	copy(dataToHash[3*frSize:], resAY[:])
	copy(dataToHash[4*frSize:], message)
	pub.HFunc.Reset()
	_, err = pub.HFunc.Write(dataToHash[:])
	if err != nil {
		return Signature{}, err
	}

	var hramInt big.Int
	hramBin := pub.HFunc.Sum([]byte{})
	hramInt.SetBytes(hramBin)

	// Compute s = randScalarInt + H(R,A,M)*S
	// going with big int to do ops mod curve order
	res.S.Mul(&hramInt, &priv.scalar).
		Add(&res.S, &randScalarInt).
		Mod(&res.S, &curveParams.Order)

	return res, nil
}

// Verify verifies an eddsa signature
// cf https://en.wikipedia.org/wiki/EdDSA
func Verify(sig Signature, message []byte, pub PublicKey) (bool, error) {

	curveParams := GetCurveParams()

	// verify that pubKey and R are on the curve
	if !pub.A.IsOnCurve() {
		return false, errNotOnCurve
	}

	// compute H(R, A, M), all parameters in data are in Montgomery form
	// compute H(R, A, M), all parameters in data are in Montgomery form
	sigRX := sig.R.X.Bytes()
	sigRY := sig.R.Y.Bytes()
	sigAX := pub.A.X.Bytes()
	sigAY := pub.A.Y.Bytes()
	sizeDataToHash := 4*frSize + len(message)
	dataToHash := make([]byte, sizeDataToHash)
	copy(dataToHash[:], sigRX[:])
	copy(dataToHash[frSize:], sigRY[:])
	copy(dataToHash[2*frSize:], sigAX[:])
	copy(dataToHash[3*frSize:], sigAY[:])
	copy(dataToHash[4*frSize:], message)
	pub.HFunc.Reset()
	_, err := pub.HFunc.Write(dataToHash[:])
	if err != nil {
		return false, err
	}

	var hramInt big.Int
	hramBin := pub.HFunc.Sum([]byte{})
	hramInt.SetBytes(hramBin)

	// lhs = cofactor*S*Base
	var lhs twistededwards.Point
	var bCofactor big.Int
	curveParams.Cofactor.ToBigInt(&bCofactor)
	lhs.ScalarMul(&curveParams.Base, &sig.S).
		ScalarMul(&lhs, &bCofactor)

	if !lhs.IsOnCurve() {
		return false, errNotOnCurve
	}

	// rhs = cofactor*(R + H(R,A,M)*A)
	var rhs twistededwards.Point
	rhs.ScalarMul(&pub.A, &hramInt).
		Add(&rhs, &sig.R).
		ScalarMul(&rhs, &bCofactor)
	if !rhs.IsOnCurve() {
		return false, errNotOnCurve
	}

	// verifies that cofactor*S*Base=cofactor*(R + H(R,A,M)*A)
	if !lhs.X.Equal(&rhs.X) || !lhs.Y.Equal(&rhs.Y) {
		return false, nil
	}
	return true, nil
}
