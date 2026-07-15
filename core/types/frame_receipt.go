// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.

package types

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// FrameReceipt is the per-frame execution result of an EIP-8141 frame
// transaction. It is exposed only in the RPC receipt JSON (the frameReceipts
// field) and is NOT part of the consensus receipt encoding.
type FrameReceipt struct {
	Status  uint64
	GasUsed uint64
	Logs    []*Log
}

type frameReceiptJSON struct {
	Status  hexutil.Uint64 `json:"status"`
	GasUsed hexutil.Uint64 `json:"gasUsed"`
	Logs    []*Log         `json:"logs"`
}

// MarshalJSON marshals a FrameReceipt as JSON, mirroring the RPC shape.
func (r FrameReceipt) MarshalJSON() ([]byte, error) {
	return json.Marshal(frameReceiptJSON{
		Status:  hexutil.Uint64(r.Status),
		GasUsed: hexutil.Uint64(r.GasUsed),
		Logs:    r.Logs,
	})
}

// UnmarshalJSON parses a FrameReceipt from the RPC JSON shape.
func (r *FrameReceipt) UnmarshalJSON(input []byte) error {
	var dec frameReceiptJSON
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	r.Status = uint64(dec.Status)
	r.GasUsed = uint64(dec.GasUsed)
	r.Logs = dec.Logs
	return nil
}
