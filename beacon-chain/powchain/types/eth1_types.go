package types

import (
	"errors"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
)

// Header represents a block header in the Ethereum blockchain.
type Header struct {
	ParentHash  common.Hash          `json:"parentHash"       gencodec:"required"`
	UncleHash   common.Hash          `json:"sha3Uncles"       gencodec:"required"`
	Coinbase    common.Address       `json:"miner"            gencodec:"required"`
	Root        common.Hash          `json:"stateRoot"        gencodec:"required"`
	TxHash      common.Hash          `json:"transactionsRoot" gencodec:"required"`
	ReceiptHash common.Hash          `json:"receiptsRoot"     gencodec:"required"`
	Bloom       gethTypes.Bloom      `json:"logsBloom"        gencodec:"required"`
	Difficulty  *big.Int             `json:"difficulty"       gencodec:"required"`
	Number      *big.Int             `json:"number"           gencodec:"required"`
	GasLimit    uint64               `json:"gasLimit"         gencodec:"required"`
	GasUsed     uint64               `json:"gasUsed"          gencodec:"required"`
	Time        uint64               `json:"timestamp"        gencodec:"required"`
	Extra       []byte               `json:"extraData"        gencodec:"required"`
	MixDigest   common.Hash          `json:"mixHash"`
	Nonce       gethTypes.BlockNonce `json:"nonce"`
	Step        *big.Int             `json:"step,omitempty"             rlp:"-"`
	Signature   []byte               `json:"signature,omitempty"        rlp:"-"`

	// BaseFee was added by EIP-1559 and is ignored in legacy headers.
	BaseFee *big.Int `json:"baseFeePerGas" rlp:"optional"`
}

// BaseHeader is a helper struct for RLP decoding
type BaseHeader struct {
	ParentHash  common.Hash     `json:"parentHash"       gencodec:"required"`
	UncleHash   common.Hash     `json:"sha3Uncles"       gencodec:"required"`
	Coinbase    common.Address  `json:"miner"            gencodec:"required"`
	Root        common.Hash     `json:"stateRoot"        gencodec:"required"`
	TxHash      common.Hash     `json:"transactionsRoot" gencodec:"required"`
	ReceiptHash common.Hash     `json:"receiptsRoot"     gencodec:"required"`
	Bloom       gethTypes.Bloom `json:"logsBloom"        gencodec:"required"`
	Difficulty  *big.Int        `json:"difficulty"       gencodec:"required"`
	Number      *big.Int        `json:"number"           gencodec:"required"`
	GasLimit    uint64          `json:"gasLimit"         gencodec:"required"`
	GasUsed     uint64          `json:"gasUsed"          gencodec:"required"`
	Time        uint64          `json:"timestamp"        gencodec:"required"`
	Extra       []byte          `json:"extraData"        gencodec:"required"`
	RawField1   rlp.RawValue    // either MixDigest or Step
	RawField2   rlp.RawValue    // either Nonce or Signature

	// BaseFee was added by EIP-1559 and is ignored in legacy headers.
	BaseFee *big.Int `json:"baseFeePerGas" rlp:"optional"`
}

func (h *Header) DecodeRLP(s *rlp.Stream) error {
	var bh BaseHeader
	if err := s.Decode(&bh); err != nil {
		return err
	}
	h.ParentHash = bh.ParentHash
	h.UncleHash = bh.UncleHash
	h.Coinbase = bh.Coinbase
	h.Root = bh.Root
	h.TxHash = bh.TxHash
	h.ReceiptHash = bh.ReceiptHash
	h.Bloom = bh.Bloom
	h.Difficulty = bh.Difficulty
	h.Number = bh.Number
	h.GasLimit = bh.GasLimit
	h.GasUsed = bh.GasUsed
	h.Time = bh.Time
	h.Extra = bh.Extra
	h.BaseFee = bh.BaseFee
	if len(bh.RawField2) > 65 {
		err := rlp.DecodeBytes(bh.RawField1, &h.Step)
		if err != nil {
			return err
		}
		err = rlp.DecodeBytes(bh.RawField2, &h.Signature)
		if err != nil {
			return err
		}
	} else {
		err := rlp.DecodeBytes(bh.RawField1, &h.MixDigest)
		if err != nil {
			return err
		}
		err = rlp.DecodeBytes(bh.RawField2, &h.Nonce)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *Header) EncodeRLP(w io.Writer) error {
	var rawField1, rawField2 rlp.RawValue
	var err error
	if len(h.Signature) > 0 {
		rawField1, err = rlp.EncodeToBytes(h.Step)
		if err != nil {
			return err
		}
		rawField2, err = rlp.EncodeToBytes(h.Signature)
		if err != nil {
			return err
		}
	} else {
		rawField1, err = rlp.EncodeToBytes(h.MixDigest)
		if err != nil {
			return err
		}
		rawField2, err = rlp.EncodeToBytes(h.Nonce)
		if err != nil {
			return err
		}
	}
	return rlp.Encode(w, BaseHeader{
		ParentHash:  h.ParentHash,
		UncleHash:   h.UncleHash,
		Coinbase:    h.Coinbase,
		Root:        h.Root,
		TxHash:      h.TxHash,
		ReceiptHash: h.ReceiptHash,
		Bloom:       h.Bloom,
		Difficulty:  h.Difficulty,
		Number:      h.Number,
		GasLimit:    h.GasLimit,
		GasUsed:     h.GasUsed,
		Time:        h.Time,
		Extra:       h.Extra,
		RawField1:   rawField1,
		RawField2:   rawField2,
		BaseFee:     h.BaseFee,
	})
}

//go:generate gencodec -type Header -field-override headerMarshaling -out gen_header_json.go

// field type overrides for gencodec
type headerMarshaling struct {
	Difficulty *hexutil.Big
	Number     *hexutil.Big
	GasLimit   hexutil.Uint64
	GasUsed    hexutil.Uint64
	Time       hexutil.Uint64
	Extra      hexutil.Bytes
	Signature  hexutil.Bytes
	BaseFee    *hexutil.Big
	Hash       common.Hash `json:"hash"` // adds call to Hash() in MarshalJSON
}

// Hash returns the block hash of the header, which is simply the keccak256 hash of its
// RLP encoding.
func (h *Header) Hash() common.Hash {
	return rlpHash(h)
}

// HeaderInfo specifies the block header information in the ETH 1.0 chain.
type HeaderInfo struct {
	Number *big.Int
	Hash   common.Hash
	Time   uint64
}

// HeaderToHeaderInfo converts an eth1 header to a header metadata type.
func HeaderToHeaderInfo(hdr *Header) (*HeaderInfo, error) {
	if hdr.Number == nil {
		// A nil number will panic when calling *big.Int.Set(...)
		return nil, errors.New("cannot convert block header with nil block number")
	}

	return &HeaderInfo{
		Hash:   hdr.Hash(),
		Number: new(big.Int).Set(hdr.Number),
		Time:   hdr.Time,
	}, nil
}

// Copy sends out a copy of the current header info.
func (h *HeaderInfo) Copy() *HeaderInfo {
	return &HeaderInfo{
		Hash:   bytesutil.ToBytes32(h.Hash[:]),
		Number: new(big.Int).Set(h.Number),
		Time:   h.Time,
	}
}
