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
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
)

// Frame is a single execution step of an EIP-8141 frame transaction.
// The six fields are the canonical tuple [mode, flags, target, gas_limit, value, data].
//
//	mode:  0 = DEFAULT, 1 = VERIFY, 2 = SENDER, 3 = POST_TX (EIP-7906)
//	flags: bit 0 = PAYMENT approval, bit 1 = EXECUTION approval, bit 2 = atomic batch
//
// Target is a pointer so contract-creation / "no target" frames encode as the RLP
// empty string, matching ethrex's `target_or_empty`.
type Frame struct {
	Mode     uint8
	Flags    uint8
	Target   *common.Address `rlp:"nil"`
	GasLimit uint64
	Value    *big.Int
	Data     []byte
}

// FrameSignature is one entry of a frame transaction's outer signature list:
// [scheme, signer, msg, signature]. Scheme 0 = secp256k1, 1 = P256.
type FrameSignature struct {
	Scheme    uint8
	Signer    common.Address
	Msg       []byte
	Signature []byte
}

// FrameTx is the EIP-8141 frame transaction (type 0x06), canonical variant.
//
// This is a DECODE-ONLY representation: go-ethereum uses it to parse and expose
// frame transactions (for explorers/tooling), not to execute, sign, or gossip
// them. Frame transactions have no single to/value/data and carry an explicit
// sender with no ECDSA signature over the transaction, so the TxData "singular"
// accessors below return decode-safe stubs; the real per-frame data is exposed
// through the Frames/Signatures/FrameSender accessors on Transaction.
//
// The field order is the canonical wire envelope
//
//	0x06 ‖ rlp([chain_id, nonce, sender, frames, signatures,
//	            max_priority_fee, max_fee, max_fee_per_blob_gas, blob_hashes])
//
// so that rlp.Encode(FrameTx) reproduces ethrex's bytes exactly and
// Transaction.Hash() == the chain's transaction hash.
type FrameTx struct {
	ChainID *big.Int
	// Canonical EIP-8141 carries a single scalar Nonce. The Hegotá variant
	// (EIP-8250) replaces it with keyed nonces (NonceKeys + NonceSeq) and appends
	// EIP-8272 recent-root references. Hegota records which envelope was decoded;
	// only the matching nonce representation is populated.
	Nonce            uint64     // canonical scalar nonce (0 when Hegota)
	NonceKeys        []*big.Int // EIP-8250 keyed nonces (nil when !Hegota)
	NonceSeq         uint64     // EIP-8250 sequence number (0 when !Hegota)
	Hegota           bool       // decoded from the 11-field Hegotá envelope
	Sender           common.Address
	Frames           []Frame
	Signatures       []FrameSignature
	GasTipCap        *big.Int // max_priority_fee_per_gas
	GasFeeCap        *big.Int // max_fee_per_gas
	MaxFeePerBlobGas *big.Int
	BlobHashes       []common.Hash
	// EIP-8272 declared recent-root references; trailing envelope field, nil when
	// !Hegota.
	RecentRootReferences []RecentRootReference
}

// RecentRootReference is an EIP-8272 declared recent-root reference — the RLP
// tuple [source_id, slot, root].
type RecentRootReference struct {
	SourceID common.Hash
	Slot     uint64
	Root     common.Hash
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *FrameTx) copy() TxData {
	cpy := &FrameTx{
		Nonce:            tx.Nonce,
		NonceSeq:         tx.NonceSeq,
		Hegota:           tx.Hegota,
		Sender:           tx.Sender,
		Frames:           make([]Frame, len(tx.Frames)),
		Signatures:       make([]FrameSignature, len(tx.Signatures)),
		BlobHashes:       make([]common.Hash, len(tx.BlobHashes)),
		ChainID:          new(big.Int),
		GasTipCap:        new(big.Int),
		GasFeeCap:        new(big.Int),
		MaxFeePerBlobGas: new(big.Int),
	}
	for i, f := range tx.Frames {
		nf := Frame{Mode: f.Mode, Flags: f.Flags, GasLimit: f.GasLimit, Value: new(big.Int)}
		nf.Target = copyAddressPtr(f.Target)
		nf.Data = common.CopyBytes(f.Data)
		if f.Value != nil {
			nf.Value.Set(f.Value)
		}
		cpy.Frames[i] = nf
	}
	for i, s := range tx.Signatures {
		cpy.Signatures[i] = FrameSignature{
			Scheme:    s.Scheme,
			Signer:    s.Signer,
			Msg:       common.CopyBytes(s.Msg),
			Signature: common.CopyBytes(s.Signature),
		}
	}
	copy(cpy.BlobHashes, tx.BlobHashes)
	if tx.NonceKeys != nil {
		cpy.NonceKeys = make([]*big.Int, len(tx.NonceKeys))
		for i, k := range tx.NonceKeys {
			cpy.NonceKeys[i] = new(big.Int)
			if k != nil {
				cpy.NonceKeys[i].Set(k)
			}
		}
	}
	if tx.RecentRootReferences != nil {
		cpy.RecentRootReferences = make([]RecentRootReference, len(tx.RecentRootReferences))
		copy(cpy.RecentRootReferences, tx.RecentRootReferences)
	}
	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
	}
	if tx.GasTipCap != nil {
		cpy.GasTipCap.Set(tx.GasTipCap)
	}
	if tx.GasFeeCap != nil {
		cpy.GasFeeCap.Set(tx.GasFeeCap)
	}
	if tx.MaxFeePerBlobGas != nil {
		cpy.MaxFeePerBlobGas.Set(tx.MaxFeePerBlobGas)
	}
	return cpy
}

// accessors for innerTx.
func (tx *FrameTx) txType() byte           { return FrameTxType }
func (tx *FrameTx) chainID() *big.Int      { return tx.ChainID }
func (tx *FrameTx) accessList() AccessList { return nil }
func (tx *FrameTx) gasFeeCap() *big.Int    { return tx.GasFeeCap }
func (tx *FrameTx) gasTipCap() *big.Int    { return tx.GasTipCap }
func (tx *FrameTx) gasPrice() *big.Int     { return tx.GasFeeCap }
// nonce reports the display nonce: for a Hegotá (EIP-8250) frame tx the account
// nonce is keyed, so surface NonceSeq (the sequence shared by the keys); the
// canonical variant uses the scalar Nonce.
func (tx *FrameTx) nonce() uint64 {
	if tx.Hegota {
		return tx.NonceSeq
	}
	return tx.Nonce
}

// data() has no single value for a frame transaction; frames carry their own data.
func (tx *FrameTx) data() []byte { return nil }

// to() is undefined for a frame transaction (each frame has its own target).
func (tx *FrameTx) to() *common.Address { return nil }

// gas() returns the summed per-frame gas limits — a display-only approximation of
// the transaction's total gas (frame transactions have no single gas field).
func (tx *FrameTx) gas() uint64 {
	var total uint64
	for _, f := range tx.Frames {
		total += f.GasLimit
	}
	return total
}

// value() returns the summed per-frame values (the total ETH the frames move),
// for display in transaction lists. Individual frame values are exposed per frame.
func (tx *FrameTx) value() *big.Int {
	total := new(big.Int)
	for _, f := range tx.Frames {
		if f.Value != nil {
			total.Add(total, f.Value)
		}
	}
	return total
}

func (tx *FrameTx) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	if baseFee == nil {
		return dst.Set(tx.GasFeeCap)
	}
	tip := dst.Sub(tx.GasFeeCap, baseFee)
	if tip.Cmp(tx.GasTipCap) > 0 {
		tip.Set(tx.GasTipCap)
	}
	return tip.Add(tip, baseFee)
}

// rawSignatureValues returns zeros: frame transactions carry no ECDSA signature
// over the transaction. The sender is explicit (see Transaction sender handling).
func (tx *FrameTx) rawSignatureValues() (v, r, s *big.Int) {
	return new(big.Int), new(big.Int), new(big.Int)
}

// setSignatureValues is a no-op for frame transactions (no signing via this path).
func (tx *FrameTx) setSignatureValues(chainID, v, r, s *big.Int) {}

func (tx *FrameTx) encode(b *bytes.Buffer) error {
	return rlp.Encode(b, tx)
}

func (tx *FrameTx) decode(input []byte) error {
	return rlp.DecodeBytes(input, tx)
}

// EncodeRLP writes the transaction's RLP envelope: the canonical 9-field
// EIP-8141 form, or the 11-field Hegotá form (EIP-8250 keyed nonces + EIP-8272
// recent-root references) when Hegota is set. Reproduces ethrex's bytes exactly,
// so Transaction.Hash() equals the chain's transaction hash.
func (tx *FrameTx) EncodeRLP(w io.Writer) error {
	if tx.Hegota {
		return rlp.Encode(w, []interface{}{
			tx.ChainID, tx.NonceKeys, tx.NonceSeq, tx.Sender, tx.Frames, tx.Signatures,
			tx.GasTipCap, tx.GasFeeCap, tx.MaxFeePerBlobGas, tx.BlobHashes, tx.RecentRootReferences,
		})
	}
	return rlp.Encode(w, []interface{}{
		tx.ChainID, tx.Nonce, tx.Sender, tx.Frames, tx.Signatures,
		tx.GasTipCap, tx.GasFeeCap, tx.MaxFeePerBlobGas, tx.BlobHashes,
	})
}

// DecodeRLP decodes either envelope, self-distinguishing by the RLP kind of the
// second field: an RLP string/byte is the canonical scalar nonce (EIP-8141); an
// RLP list is the Hegotá keyed-nonce vector (EIP-8250), which is followed by a
// nonce_seq field and, at the end of the envelope, a recent_root_references list
// (EIP-8272).
func (tx *FrameTx) DecodeRLP(s *rlp.Stream) error {
	if _, err := s.List(); err != nil {
		return err
	}
	if err := s.Decode(&tx.ChainID); err != nil {
		return err
	}
	kind, _, err := s.Kind()
	if err != nil {
		return err
	}
	if kind == rlp.List {
		tx.Hegota = true
		if err := s.Decode(&tx.NonceKeys); err != nil {
			return err
		}
		if err := s.Decode(&tx.NonceSeq); err != nil {
			return err
		}
	} else if err := s.Decode(&tx.Nonce); err != nil {
		return err
	}
	if err := s.Decode(&tx.Sender); err != nil {
		return err
	}
	if err := s.Decode(&tx.Frames); err != nil {
		return err
	}
	if err := s.Decode(&tx.Signatures); err != nil {
		return err
	}
	if err := s.Decode(&tx.GasTipCap); err != nil {
		return err
	}
	if err := s.Decode(&tx.GasFeeCap); err != nil {
		return err
	}
	if err := s.Decode(&tx.MaxFeePerBlobGas); err != nil {
		return err
	}
	if err := s.Decode(&tx.BlobHashes); err != nil {
		return err
	}
	if tx.Hegota {
		if err := s.Decode(&tx.RecentRootReferences); err != nil {
			return err
		}
	}
	return s.ListEnd()
}

// sigHash is the EIP-8141 signature hash: keccak(0x06 ‖ rlp(envelope)) with the
// signature bytes of empty-msg signatures elided (they sign this hash, not their
// own bytes). Frame transactions resolve sender from the explicit field, so this
// is provided for completeness / verification, not for sender recovery.
func (tx *FrameTx) sigHash(chainID *big.Int) common.Hash {
	elided := tx.copy().(*FrameTx)
	for i := range elided.Signatures {
		if len(elided.Signatures[i].Msg) == 0 {
			elided.Signatures[i].Signature = nil
		}
	}
	return prefixedRlpHash(FrameTxType, elided)
}

// Frames returns the frames of a frame transaction, or nil for other tx types.
func (tx *Transaction) Frames() []Frame {
	if inner, ok := tx.inner.(*FrameTx); ok {
		return inner.Frames
	}
	return nil
}

// FrameSignatures returns the outer signature list of a frame transaction, or nil.
func (tx *Transaction) FrameSignatures() []FrameSignature {
	if inner, ok := tx.inner.(*FrameTx); ok {
		return inner.Signatures
	}
	return nil
}

// FrameSender returns the explicit sender of a frame transaction. The bool is
// false for non-frame transactions.
func (tx *Transaction) FrameSender() (common.Address, bool) {
	if inner, ok := tx.inner.(*FrameTx); ok {
		return inner.Sender, true
	}
	return common.Address{}, false
}

// IsFrameHegota reports whether a frame transaction was decoded from the Hegotá
// (EIP-8250 keyed-nonce) envelope rather than the canonical EIP-8141 one.
func (tx *Transaction) IsFrameHegota() bool {
	inner, ok := tx.inner.(*FrameTx)
	return ok && inner.Hegota
}

// FrameNonceKeys returns a Hegotá frame transaction's EIP-8250 keyed nonces, or
// nil for a canonical (scalar-nonce) frame tx or a non-frame tx.
func (tx *Transaction) FrameNonceKeys() []*big.Int {
	if inner, ok := tx.inner.(*FrameTx); ok {
		return inner.NonceKeys
	}
	return nil
}

// FrameNonceSeq returns a Hegotá frame transaction's EIP-8250 sequence number;
// the bool is false for a canonical frame tx or a non-frame tx.
func (tx *Transaction) FrameNonceSeq() (uint64, bool) {
	if inner, ok := tx.inner.(*FrameTx); ok && inner.Hegota {
		return inner.NonceSeq, true
	}
	return 0, false
}

// FrameRecentRootReferences returns a frame transaction's EIP-8272 recent-root
// references, or nil.
func (tx *Transaction) FrameRecentRootReferences() []RecentRootReference {
	if inner, ok := tx.inner.(*FrameTx); ok {
		return inner.RecentRootReferences
	}
	return nil
}

// JSON representations mirroring the RPC shape of a frame transaction, used by
// Transaction's Marshal/UnmarshalJSON (the ethclient path).

type frameJSON struct {
	Mode     hexutil.Uint64  `json:"mode"`
	Flags    hexutil.Uint64  `json:"flags"`
	To       *common.Address `json:"to"`
	GasLimit hexutil.Uint64  `json:"gasLimit"`
	Value    *hexutil.Big    `json:"value"`
	Data     hexutil.Bytes   `json:"data"`
}

type frameSignatureJSON struct {
	Scheme    hexutil.Uint64 `json:"scheme"`
	Signer    common.Address `json:"signer"`
	Msg       hexutil.Bytes  `json:"msg"`
	Signature hexutil.Bytes  `json:"signature"`
}

func framesToJSON(frames []Frame) []frameJSON {
	out := make([]frameJSON, len(frames))
	for i, f := range frames {
		out[i] = frameJSON{
			Mode:     hexutil.Uint64(f.Mode),
			Flags:    hexutil.Uint64(f.Flags),
			To:       f.Target,
			GasLimit: hexutil.Uint64(f.GasLimit),
			Value:    (*hexutil.Big)(f.Value),
			Data:     f.Data,
		}
	}
	return out
}

func framesFromJSON(in []frameJSON) []Frame {
	out := make([]Frame, len(in))
	for i, f := range in {
		v := (*big.Int)(f.Value)
		if v == nil {
			v = new(big.Int)
		}
		out[i] = Frame{
			Mode:     uint8(f.Mode),
			Flags:    uint8(f.Flags),
			Target:   f.To,
			GasLimit: uint64(f.GasLimit),
			Value:    v,
			Data:     f.Data,
		}
	}
	return out
}

func signaturesToJSON(sigs []FrameSignature) []frameSignatureJSON {
	out := make([]frameSignatureJSON, len(sigs))
	for i, s := range sigs {
		out[i] = frameSignatureJSON{
			Scheme:    hexutil.Uint64(s.Scheme),
			Signer:    s.Signer,
			Msg:       s.Msg,
			Signature: s.Signature,
		}
	}
	return out
}

func signaturesFromJSON(in []frameSignatureJSON) []FrameSignature {
	out := make([]FrameSignature, len(in))
	for i, s := range in {
		out[i] = FrameSignature{
			Scheme:    uint8(s.Scheme),
			Signer:    s.Signer,
			Msg:       s.Msg,
			Signature: s.Signature,
		}
	}
	return out
}

// recentRootRefJSON is the RPC JSON shape of an EIP-8272 recent-root reference.
type recentRootRefJSON struct {
	SourceID common.Hash    `json:"sourceId"`
	Slot     hexutil.Uint64 `json:"slot"`
	Root     common.Hash    `json:"root"`
}

func recentRootRefsToJSON(refs []RecentRootReference) []recentRootRefJSON {
	if refs == nil {
		return nil
	}
	out := make([]recentRootRefJSON, len(refs))
	for i, r := range refs {
		out[i] = recentRootRefJSON{SourceID: r.SourceID, Slot: hexutil.Uint64(r.Slot), Root: r.Root}
	}
	return out
}

func recentRootRefsFromJSON(in []recentRootRefJSON) []RecentRootReference {
	if in == nil {
		return nil
	}
	out := make([]RecentRootReference, len(in))
	for i, r := range in {
		out[i] = RecentRootReference{SourceID: r.SourceID, Slot: uint64(r.Slot), Root: r.Root}
	}
	return out
}
