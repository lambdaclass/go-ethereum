package types

import "testing"

// Regression: ethrex omits transactionHash on receipt logs; the fork's
// Log.UnmarshalJSON must tolerate that so frame-tx receipts (which emit a
// transfer log) parse — otherwise the explorer loses all block/gas/status data.
func TestEthrexFrameReceiptParses(t *testing.T) {
	const j = `{"blockHash":"0x1164665365a4747612fe09f3d3666957ee70e98bc4a55b2ab7c74d6e1bbaacc1","blockNumber":"0x3638b","contractAddress":null,"cumulativeGasUsed":"0x5501","effectiveGasPrice":"0x3b9aca07","frameReceipts":[{"gasUsed":"0x0","logs":[],"status":"0x1"},{"gasUsed":"0x0","logs":[{"address":"0xfffffffffffffffffffffffffffffffffffffffe","data":"0x0000000000000000000000000000000000000000000000000000000000000001","topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x0000000000000000000000008943545177806ed17b9f23f0a21ee5948ecaa776","0x00000000000000000000000000000000000000000000000000000000deadbeef"]}],"status":"0x1"},{"gasUsed":"0x0","logs":[],"status":"0x1"}],"from":"0x8943545177806ed17b9f23f0a21ee5948ecaa776","gasUsed":"0x5501","logs":[{"address":"0xfffffffffffffffffffffffffffffffffffffffe","blockHash":"0x1164665365a4747612fe09f3d3666957ee70e98bc4a55b2ab7c74d6e1bbaacc1","blockNumber":"0x3638b","data":"0x0000000000000000000000000000000000000000000000000000000000000001","logIndex":"0x0","removed":false,"topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x0000000000000000000000008943545177806ed17b9f23f0a21ee5948ecaa776","0x00000000000000000000000000000000000000000000000000000000deadbeef"],"transactionHash":"0x1aea2d699a977fbbe948f48746945f024dd8775cf67863ab52c015e7f8867668","transactionIndex":"0x0"}],"logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000008000000000000000000000000000008000000000000000000000000000000000000000000008020000000002000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000002000000000008000000000000000000000000000000000000000000400000000000000000000000000000000000800000000000000000000000000000","payer":"0x8943545177806ed17b9f23f0a21ee5948ecaa776","status":"0x1","to":"0x8943545177806ed17b9f23f0a21ee5948ecaa776","transactionHash":"0x1aea2d699a977fbbe948f48746945f024dd8775cf67863ab52c015e7f8867668","transactionIndex":"0x0","type":"0x6"}`
	var r Receipt
	if err := r.UnmarshalJSON([]byte(j)); err != nil {
		t.Fatalf("frame receipt must parse (ethrex omits log transactionHash): %v", err)
	}
	if r.Status != 1 || r.GasUsed == 0 || r.BlockNumber == nil || r.BlockNumber.Sign() == 0 {
		t.Fatalf("receipt fields not populated: status=%d gas=%d block=%v", r.Status, r.GasUsed, r.BlockNumber)
	}
	if len(r.FrameReceipts) != 3 || r.Payer == nil {
		t.Fatalf("frame receipt extras missing: frameReceipts=%d payer=%v", len(r.FrameReceipts), r.Payer)
	}
}
