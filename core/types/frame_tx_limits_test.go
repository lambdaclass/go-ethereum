// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.

package types

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// The two constants below are the same transaction from ethrex's own golden test
// (crates/common/types/transaction.rs), once per frame wire form: the original
// scalar `gas_limit` in the gas slot, and the current `limits = [execution,
// state]`. ethrex locks both because a chain that produced frames before the
// switch must keep re-encoding them the old way to serve its own history.
const (
	frameGasScalarGoldenRLP = "f8ae01c1800794000000000000000000000000000000000000abcde8ca01038082520880821122dc0280940000000000000000000000000000000000001234829c408080f85cf85a0194000000000000000000000000000000000000abcd80b8410101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101843b9aca008506fc23ac0080c0c0"
	frameGasLimitsGoldenRLP = "f8b201c1800794000000000000000000000000000000000000abcdeccc010380c48252088080821122de0280940000000000000000000000000000000000001234c4829c40808080f85cf85a0194000000000000000000000000000000000000abcd80b8410101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101843b9aca008506fc23ac0080c0c0"
)

// A real transaction from the Hegotá devnet, block 292867, produced after that
// chain activated the `limits` form. Its hash is the one the chain reports, so a
// decoder that re-encodes these bytes into anything else fails this test.
const (
	hegotaLimitsRawTx  = "06f8d583301824c1800194ea4e13a1107b002aa7547df6681b80ecc191b13ef84be2010394ea4e13a1107b002aa7547df6681b80ecc191b13ec8830186a08307a1208080e7028094a5208a807db219b640ba2a77a2719e5a0a43d024c8830186a08307a12085e8d4a5100080f85cf85a0194ea4e13a1107b002aa7547df6681b80ecc191b13e80b84100ee36949779bbf1fa62b182307b74dfa7795f356980e06d502a99f44f07d4453704abf269afe11fbc9b8c2a68a21f5e6282c812d923858dc6d92b85a7640d140e843b9aca008502540be40080c0c0"
	hegotaLimitsTxHash = "0x759f6fe556eea6c50150f6e405ee10afce4aaaf16f1aa7ab770facc71dd41faf"
)

func TestFrameTxDecodesBothGasForms(t *testing.T) {
	for _, tc := range []struct {
		name       string
		rlp        string
		wantLimits bool
		wantGas    []uint64
		wantState  []uint64
	}{
		{"scalar", frameGasScalarGoldenRLP, false, []uint64{0x5208, 0x9c40}, []uint64{0, 0}},
		{"limits", frameGasLimitsGoldenRLP, true, []uint64{0x5208, 0x9c40}, []uint64{0, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := hex.DecodeString("06" + tc.rlp)
			if err != nil {
				t.Fatalf("bad test vector: %v", err)
			}
			var tx Transaction
			if err := tx.UnmarshalBinary(raw); err != nil {
				t.Fatalf("decode: %v", err)
			}
			frames := tx.Frames()
			if len(frames) != len(tc.wantGas) {
				t.Fatalf("got %d frames, want %d", len(frames), len(tc.wantGas))
			}
			for i, f := range frames {
				if f.Limits != tc.wantLimits {
					t.Errorf("frame %d: Limits = %v, want %v", i, f.Limits, tc.wantLimits)
				}
				if f.GasLimit != tc.wantGas[i] {
					t.Errorf("frame %d: GasLimit = %d, want %d", i, f.GasLimit, tc.wantGas[i])
				}
				if f.StateGasLimit != tc.wantState[i] {
					t.Errorf("frame %d: StateGasLimit = %d, want %d", i, f.StateGasLimit, tc.wantState[i])
				}
			}
			// Re-encoding must reproduce the input: the transaction hash and the
			// transactions root are derived from it.
			back, err := tx.MarshalBinary()
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if !bytes.Equal(back, raw) {
				t.Errorf("re-encoded to %x, want %x", back, raw)
			}
		})
	}
}

func TestFrameTxLimitsRealTransaction(t *testing.T) {
	raw, err := hex.DecodeString(hegotaLimitsRawTx)
	if err != nil {
		t.Fatalf("bad test vector: %v", err)
	}
	var tx Transaction
	if err := tx.UnmarshalBinary(raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got := tx.Hash(); got != common.HexToHash(hegotaLimitsTxHash) {
		t.Fatalf("hash = %s, want %s", got.Hex(), hegotaLimitsTxHash)
	}
	frames := tx.Frames()
	if len(frames) != 2 {
		t.Fatalf("got %d frames, want 2", len(frames))
	}
	for i, f := range frames {
		if !f.Limits {
			t.Errorf("frame %d: Limits = false, want true", i)
		}
		if f.GasLimit != 0x186a0 {
			t.Errorf("frame %d: GasLimit = %d, want %d", i, f.GasLimit, 0x186a0)
		}
		if f.StateGasLimit != 0x7a120 {
			t.Errorf("frame %d: StateGasLimit = %d, want %d", i, f.StateGasLimit, 0x7a120)
		}
	}
	back, err := tx.MarshalBinary()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if !bytes.Equal(back, raw) {
		t.Errorf("re-encoded to %x, want %x", back, raw)
	}
}
