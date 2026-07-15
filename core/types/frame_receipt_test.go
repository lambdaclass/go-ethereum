// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.

package types

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestFrameReceiptJSON verifies the RPC-JSON round-trip of the frame-receipt
// fields and that standard receipts are unaffected (no frame fields emitted).
func TestFrameReceiptJSON(t *testing.T) {
	payer := common.HexToAddress("0x8943545177806ed17b9f23f0a21ee5948ecaa776")
	r := &Receipt{
		Type:              FrameTxType,
		Status:            1,
		CumulativeGasUsed: 0x52e2,
		GasUsed:           0x52e2,
		EffectiveGasPrice: big.NewInt(7),
		TxHash:            common.HexToHash("0x73095c92dedf601c499c8263199f21a2b998c2b4afef712f93c5114ff3908089"),
		Logs:              []*Log{},
		FrameReceipts: []*FrameReceipt{
			{Status: 1, GasUsed: 0, Logs: []*Log{}},
			{Status: 1, GasUsed: 0, Logs: []*Log{}},
		},
		Payer: &payer,
	}

	js, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if !strings.Contains(string(js), `"frameReceipts"`) || !strings.Contains(string(js), `"payer"`) {
		t.Fatalf("frame fields missing from receipt JSON: %s", js)
	}

	var r2 Receipt
	if err := r2.UnmarshalJSON(js); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if len(r2.FrameReceipts) != 2 {
		t.Fatalf("frameReceipts = %d, want 2", len(r2.FrameReceipts))
	}
	if r2.FrameReceipts[0].Status != 1 {
		t.Fatalf("frameReceipts[0].Status = %d, want 1", r2.FrameReceipts[0].Status)
	}
	if r2.Payer == nil || *r2.Payer != payer {
		t.Fatalf("payer = %v, want %v", r2.Payer, payer)
	}

	// A standard receipt must not emit frame fields (omitempty), keeping the
	// RPC shape identical to upstream for non-frame transactions.
	std := &Receipt{Type: DynamicFeeTxType, Status: 1, CumulativeGasUsed: 1, GasUsed: 1, EffectiveGasPrice: big.NewInt(1), Logs: []*Log{}}
	sjs, err := std.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON(std): %v", err)
	}
	if strings.Contains(string(sjs), "frameReceipts") || strings.Contains(string(sjs), "payer") {
		t.Fatalf("standard receipt leaked frame fields: %s", sjs)
	}
}
