// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.

package types

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// A real Hegotá (EIP-8250 keyed-nonce) frame transaction pulled from the live
// hegota devnet via eth_getTransactionByHash (block 0x340ac). Its keyed nonce is
// key 0x309 (777) / seq 0, two frames (VERIFY + SENDER), self-paid.
const hegotaFrameTxJSON = `{"blobVersionedHashes":[],"blockHash":"0xbd4d98aa24669ceeed915e2c8b8b6e6ceb77ee00c2da20dca63dbc78649b8b79","blockNumber":"0x340ac","chainId":"0x301824","frames":[{"data":"0x","flags":"0x3","gasLimit":"0x13880","mode":"0x1","to":"0x8943545177806ed17b9f23f0a21ee5948ecaa776","value":"0x0"},{"data":"0x","flags":"0x0","gasLimit":"0x7530","mode":"0x2","to":"0x00000000000000000000000000000000deadbeef","value":"0x1"}],"from":"0x8943545177806ed17b9f23f0a21ee5948ecaa776","hash":"0x5c00db377a178ec6ebd5fd8e7414f40f9ffaf7ded8066909f29ed6a691c9df4c","maxFeePerBlobGas":"0x0","maxFeePerGas":"0x3b9aca0e","maxPriorityFeePerGas":"0x3b9aca00","nonceKeys":["0x309"],"nonceSeq":"0x0","recentRootReferences":[],"sender":"0x8943545177806ed17b9f23f0a21ee5948ecaa776","signatures":[{"msg":"0x","scheme":"0x0","signature":"0x1c1e153452cb8146920b5b3c152de9a450981d82a215d15b3b57ea73309ea3a07967678e7a03e6c0bab7716ef6c820bb7b544035be0a8b96cd2f16f968b9213ec0","signer":"0x8943545177806ed17b9f23f0a21ee5948ecaa776"}],"transactionIndex":"0x0","type":"0x6"}`

func TestHegotaFrameTxJSONAndHash(t *testing.T) {
	wantHash := common.HexToHash("0x5c00db377a178ec6ebd5fd8e7414f40f9ffaf7ded8066909f29ed6a691c9df4c")

	var tx Transaction
	if err := tx.UnmarshalJSON([]byte(hegotaFrameTxJSON)); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if tx.Type() != FrameTxType {
		t.Fatalf("type = %d, want %d", tx.Type(), FrameTxType)
	}
	if !tx.IsFrameHegota() {
		t.Fatal("IsFrameHegota() = false, want true for a keyed-nonce frame tx")
	}
	keys := tx.FrameNonceKeys()
	if len(keys) != 1 || keys[0].Cmp(big.NewInt(0x309)) != 0 {
		t.Fatalf("FrameNonceKeys() = %v, want [777]", keys)
	}
	if seq, ok := tx.FrameNonceSeq(); !ok || seq != 0 {
		t.Fatalf("FrameNonceSeq() = (%d, %v), want (0, true)", seq, ok)
	}
	if len(tx.Frames()) != 2 {
		t.Fatalf("Frames() len = %d, want 2", len(tx.Frames()))
	}
	if refs := tx.FrameRecentRootReferences(); len(refs) != 0 {
		t.Fatalf("FrameRecentRootReferences() len = %d, want 0", len(refs))
	}

	// Byte-exact encode: hashing 0x06 ‖ rlp(hegota envelope) must reproduce the
	// on-chain transaction hash. This proves EncodeRLP matches ethrex's bytes.
	if got := tx.Hash(); got != wantHash {
		t.Fatalf("Hash() = %s, want %s (hegota envelope encode is not byte-exact)", got, wantHash)
	}

	// RLP round-trip: encode → decode → re-encode must be identical and preserve
	// the hegota shape + hash.
	raw, err := tx.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var tx2 Transaction
	if err := tx2.UnmarshalBinary(raw); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if !tx2.IsFrameHegota() {
		t.Fatal("RLP round-trip lost the hegota shape")
	}
	if tx2.Hash() != wantHash {
		t.Fatalf("round-trip Hash() = %s, want %s", tx2.Hash(), wantHash)
	}
	raw2, err := tx2.MarshalBinary()
	if err != nil {
		t.Fatalf("re-MarshalBinary: %v", err)
	}
	if !bytes.Equal(raw, raw2) {
		t.Fatalf("RLP round-trip mismatch:\n first=%x\n  again=%x", raw, raw2)
	}
	keys2 := tx2.FrameNonceKeys()
	if len(keys2) != 1 || keys2[0].Cmp(big.NewInt(0x309)) != 0 {
		t.Fatalf("round-trip FrameNonceKeys() = %v, want [777]", keys2)
	}
}
