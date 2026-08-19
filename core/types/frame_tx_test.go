// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.

package types

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// Live canonical EIP-8141 (single-nonce) frame transaction mined on the srv8
// eip8141 devnet: a self-verified transfer from 0x8943… (VERIFY approve
// exec+payment, then SENDER transfer of 1 wei). Used as the byte-exact fixture.
const (
	srv8FrameRawHex = "0x06f8c28330182437948943545177806ed17b9f23f0a21ee5948ecaa776" +
		"f83bdd0103948943545177806ed17b9f23f0a21ee5948ecaa776830138808080" +
		"dc0280948943545177806ed17b9f23f0a21ee5948ecaa7768275300180" +
		"f85cf85a80948943545177806ed17b9f23f0a21ee5948ecaa77680b841" +
		"1c5a901675d5b1e07f42e7bd5aa2b8994949aa755c41601bdd7595d539c78ed2" +
		"c906ddf92faea5157256dcd6848cd4b4cfb18551ea36ea5c714f0053a8f21cd06d" +
		"847735940085174876e80080c0"
	srv8FrameHash    = "0x73095c92dedf601c499c8263199f21a2b998c2b4afef712f93c5114ff3908089"
	srv8FrameSigHash = "0xfbc14c1b2dac2e81b216847bca6d76c5d822848e3eb8823e742ceb6ed90acfa7"
)

// TestFrameTxDecodeCanonical verifies that the fork decodes a live canonical
// frame transaction, round-trips it byte-for-byte, and reproduces the chain's
// transaction hash and sig_hash — the correctness gate for Dora.
func TestFrameTxDecodeCanonical(t *testing.T) {
	raw := common.FromHex(srv8FrameRawHex)

	var tx Transaction
	if err := tx.UnmarshalBinary(raw); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	if tx.Type() != FrameTxType {
		t.Fatalf("type = 0x%x, want 0x%x", tx.Type(), FrameTxType)
	}

	// Byte-exact round-trip.
	got, err := tx.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if !bytes.Equal(got, raw) {
		t.Fatalf("round-trip mismatch:\n got  %x\n want %x", got, raw)
	}

	// Hash parity with the chain (keccak(0x06 ‖ rlp(inner))).
	if h := tx.Hash().Hex(); h != srv8FrameHash {
		t.Fatalf("Hash() = %s, want %s", h, srv8FrameHash)
	}

	// sig_hash parity (empty-msg signature bytes elided).
	inner := tx.inner.(*FrameTx)
	if sh := inner.sigHash(big.NewInt(3151908)).Hex(); sh != srv8FrameSigHash {
		t.Fatalf("sigHash = %s, want %s", sh, srv8FrameSigHash)
	}

	// Decoded structure.
	sender := common.HexToAddress("0x8943545177806ed17b9f23f0a21ee5948ecaa776")
	if s, ok := tx.FrameSender(); !ok || s != sender {
		t.Fatalf("FrameSender = %v (ok=%v), want %v", s, ok, sender)
	}
	if inner.Nonce != 0x37 {
		t.Fatalf("nonce = %d, want 55", inner.Nonce)
	}
	frames := tx.Frames()
	if len(frames) != 2 {
		t.Fatalf("frames = %d, want 2", len(frames))
	}
	if frames[0].Mode != 1 || frames[0].Flags != 3 {
		t.Fatalf("frame0 = mode %d flags %d, want VERIFY(1)/0x3", frames[0].Mode, frames[0].Flags)
	}
	if frames[1].Mode != 2 || frames[1].Value.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("frame1 = mode %d value %s, want SENDER(2)/1", frames[1].Mode, frames[1].Value)
	}
	// Signer is optional on the wire; this fixture names it, so it must decode to
	// the address rather than to nil.
	if sigs := tx.FrameSignatures(); len(sigs) != 1 || sigs[0].Scheme != 0 || sigs[0].Signer == nil || *sigs[0].Signer != sender {
		t.Fatalf("signatures = %+v, want one sig naming the sender", tx.FrameSignatures())
	}

	// JSON round-trip (the ethclient path Dora uses via BlockByNumber).
	js, err := tx.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	var tx2 Transaction
	if err := tx2.UnmarshalJSON(js); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if tx2.Hash() != tx.Hash() {
		t.Fatalf("JSON round-trip hash mismatch: %s vs %s", tx2.Hash().Hex(), tx.Hash().Hex())
	}
	if s, ok := tx2.FrameSender(); !ok || s != sender {
		t.Fatalf("JSON round-trip sender = %v (ok=%v), want %v", s, ok, sender)
	}
}
