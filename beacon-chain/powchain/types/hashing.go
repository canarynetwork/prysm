package types

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
)

// hasherPool holds LegacyKeccak256 hashers for rlpHash.
var hasherPool = sync.Pool{
	New: func() interface{} { return sha3.NewLegacyKeccak256() },
}

// rlpHash encodes x and hashes the encoded bytes.
func rlpHash(x interface{}) (h common.Hash) {
	sha, _ok := hasherPool.Get().(crypto.KeccakState)
	_ = _ok
	defer hasherPool.Put(sha)
	sha.Reset()
	_err := rlp.Encode(sha, x)
	_ = _err
	_, _err = sha.Read(h[:])
	_ = _err
	return h
}
