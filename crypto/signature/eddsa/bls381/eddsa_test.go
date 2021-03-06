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
	"testing"

	"github.com/consensys/gnark/crypto/hash/mimc/bls381"
	"github.com/consensys/gnark/crypto/hash/mimc/bn256"
	"github.com/consensys/gurvy/bls381/fr"
)

func TestEddsa(t *testing.T) {

	var seed [32]byte
	s := []byte("eddsa")
	for i, v := range s {
		seed[i] = v
	}

	hFunc := bn256.NewMiMC("seed")

	// create eddsa obj and sign a message
	pubKey, privKey := New(seed, hFunc)
	var frMsg fr.Element
	frMsg.SetString("44717650746155748460101257525078853138837311576962212923649547644148297035978")
	msgBin := frMsg.Bytes()
	signature, err := Sign(msgBin[:], pubKey, privKey)
	if err != nil {
		t.Fatal(err)
	}

	// verifies correct msg
	res, err := Verify(signature, msgBin[:], pubKey)
	if err != nil {
		t.Fatal(err)
	}
	if !res {
		t.Fatal("Verifiy correct signature should return true")
	}

	// verifies wrong msg
	frMsg.SetString("44717650746155748460101257525078853138837311576962212923649547644148297035979")
	msgBin = frMsg.Bytes()
	res, err = Verify(signature, msgBin[:], pubKey)
	if err != nil {
		t.Fatal(err)
	}
	if res {
		t.Fatal("Verfiy wrong signature should be false")
	}

}

// benchmarks

func BenchmarkVerify(b *testing.B) {

	var seed [32]byte
	s := []byte("eddsa")
	for i, v := range s {
		seed[i] = v
	}

	hFunc := bls381.NewMiMC("seed")

	// create eddsa obj and sign a message
	pubKey, privKey := New(seed, hFunc)
	var frMsg fr.Element
	frMsg.SetString("44717650746155748460101257525078853138837311576962212923649547644148297035978")
	msgBin := frMsg.Bytes()
	signature, _ := Sign(msgBin[:], pubKey, privKey)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Verify(signature, msgBin[:], pubKey)
	}
}
