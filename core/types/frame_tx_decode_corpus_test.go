// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.

package types

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// The corpus in testdata/frame_tx_decode_corpus.json exercises every optional or
// variable-width field of the EIP-8141 envelope, because a fixed-width Go type
// standing in for an optional wire field rejects the whole transaction rather
// than degrading -- which is how an absent signature signer went unnoticed until
// transactions on a live chain silently failed to index.
//
// Provenance, which is what makes the corpus evidence rather than assertion:
//
//   - The first case is a real transaction from the Hegotá testnet (chain 8141),
//     hash 0xf803a6d8..., carrying a SECP256K1 signature with no signer.
//   - Every other case was encoded with ethrex's own encoder and then offered to a
//     live ethrex node through ethrex_simulateFrameTransaction, which runs
//     Transaction::decode_canonical before judging anything. Only envelopes Rust
//     accepted are included, so a failure here is this decoder disagreeing with
//     the reference implementation, not a malformed fixture.
//
// Each recorded hash is keccak256 of the raw bytes, computed independently of this
// package, so asserting Hash() equality checks decode *and* re-encode: a field
// read into the wrong slot changes the hash even when decoding reports success.
type frameDecodeCase struct {
	Name string `json:"name"`
	Raw  string `json:"raw"`
	Hash string `json:"hash"`
}

func loadFrameDecodeCorpus(t *testing.T) []frameDecodeCase {
	t.Helper()
	blob, err := os.ReadFile("testdata/frame_tx_decode_corpus.json")
	if err != nil {
		t.Fatalf("reading corpus: %v", err)
	}
	var cases []frameDecodeCase
	if err := json.Unmarshal(blob, &cases); err != nil {
		t.Fatalf("parsing corpus: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("corpus is empty")
	}
	return cases
}

func TestFrameTxDecodeCorpus(t *testing.T) {
	for _, tc := range loadFrameDecodeCorpus(t) {
		t.Run(tc.Name, func(t *testing.T) {
			raw, err := hex.DecodeString(tc.Raw)
			if err != nil {
				t.Fatalf("bad hex in corpus: %v", err)
			}
			var tx Transaction
			if err := tx.UnmarshalBinary(raw); err != nil {
				t.Fatalf("decode failed: %v", err)
			}
			if tx.Type() != FrameTxType {
				t.Fatalf("type = %d, want %d", tx.Type(), FrameTxType)
			}
			if got := tx.Hash().Hex(); got != tc.Hash {
				t.Fatalf("re-encode changed the transaction: hash = %s, want %s", got, tc.Hash)
			}
		})
	}
}

// TestFrameTxSignerOptional pins the semantics the corpus exercises: the signer is
// absent or present on the wire, and an absent one must survive as nil rather than
// as the zero address, which is a real address and would misreport the signer.
func TestFrameTxSignerOptional(t *testing.T) {
	byName := map[string]string{}
	for _, tc := range loadFrameDecodeCorpus(t) {
		byName[tc.Name] = tc.Raw
	}
	want := map[string]bool{ // case name -> signer expected to be present
		"chain_tx_signer_absent_secp256k1": false,
		"signer_absent_secp256k1":          false,
		"signer_named_secp256k1":           true,
		"signer_absent_arbitrary":          false,
		"signer_absent_p256":               false,
		"signer_named_p256":                true,
	}
	for name, present := range want {
		raw, ok := byName[name]
		if !ok {
			t.Fatalf("corpus is missing the %q case", name)
		}
		blob, err := hex.DecodeString(raw)
		if err != nil {
			t.Fatalf("%s: bad hex: %v", name, err)
		}
		var tx Transaction
		if err := tx.UnmarshalBinary(blob); err != nil {
			t.Fatalf("%s: decode failed: %v", name, err)
		}
		sigs := tx.FrameSignatures()
		if len(sigs) == 0 {
			t.Fatalf("%s: no signatures decoded", name)
		}
		if got := sigs[0].Signer != nil; got != present {
			t.Errorf("%s: signer present = %v, want %v", name, got, present)
		}
		if !present && sigs[0].Signer != nil {
			t.Errorf("%s: absent signer decoded as %s", name, sigs[0].Signer.Hex())
		}
	}

	// A named signer must round-trip to the address on the wire, not merely be non-nil.
	blob, _ := hex.DecodeString(byName["signer_named_secp256k1"])
	var tx Transaction
	if err := tx.UnmarshalBinary(blob); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	wantAddr := common.HexToAddress("0xe3b43d9c13be09a4c3ab36b21cfe5b268af9b5b9")
	if got := tx.FrameSignatures()[0].Signer; got == nil || *got != wantAddr {
		t.Errorf("named signer = %v, want %s", got, wantAddr.Hex())
	}

	// The mixed list proves the two forms coexist inside one transaction.
	blob, _ = hex.DecodeString(byName["mixed_signer_list"])
	var mixed Transaction
	if err := mixed.UnmarshalBinary(blob); err != nil {
		t.Fatalf("mixed_signer_list decode failed: %v", err)
	}
	sigs := mixed.FrameSignatures()
	if len(sigs) != 2 {
		t.Fatalf("mixed_signer_list: %d signatures, want 2", len(sigs))
	}
	if sigs[0].Signer != nil || sigs[1].Signer == nil {
		t.Errorf("mixed_signer_list: got signers %v and %v, want absent then present", sigs[0].Signer, sigs[1].Signer)
	}
}

// TestFrameTxSignerJSONNull keeps the JSON view honest: an absent signer renders as
// null. Rendering it as the zero address would put a plausible-looking address in
// an explorer for a field the transaction never carried.
func TestFrameTxSignerJSONNull(t *testing.T) {
	byName := map[string]string{}
	for _, tc := range loadFrameDecodeCorpus(t) {
		byName[tc.Name] = tc.Raw
	}
	for name, wantSigner := range map[string]any{
		"chain_tx_signer_absent_secp256k1": nil,
		"signer_named_secp256k1":           "0xe3b43d9c13be09a4c3ab36b21cfe5b268af9b5b9",
	} {
		blob, _ := hex.DecodeString(byName[name])
		var tx Transaction
		if err := tx.UnmarshalBinary(blob); err != nil {
			t.Fatalf("%s: decode failed: %v", name, err)
		}
		encoded, err := json.Marshal(&tx)
		if err != nil {
			t.Fatalf("%s: marshal failed: %v", name, err)
		}
		var view struct {
			Signatures []struct {
				Signer *string `json:"signer"`
			} `json:"signatures"`
		}
		if err := json.Unmarshal(encoded, &view); err != nil {
			t.Fatalf("%s: unmarshal failed: %v", name, err)
		}
		if len(view.Signatures) == 0 {
			t.Fatalf("%s: JSON carried no signatures", name)
		}
		got := view.Signatures[0].Signer
		if wantSigner == nil {
			if got != nil {
				t.Errorf("%s: signer rendered as %q, want null", name, *got)
			}
			continue
		}
		if got == nil || *got != wantSigner.(string) {
			t.Errorf("%s: signer rendered as %v, want %v", name, got, wantSigner)
		}
	}
}
