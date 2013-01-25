// Copyright (c) 2012, Suryandaru Triandana <syndtr@gmail.com>
// All rights reserved.
//
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// This LevelDB Go implementation is based on LevelDB C++ implementation.
// Which contains the following header:
//   Copyright (c) 2011 The LevelDB Authors. All rights reserved.
//   Use of this source code is governed by a BSD-style license that can be
//   found in the LEVELDBCPP_LICENSE file. See the LEVELDBCPP_AUTHORS file
//   for names of contributors.

package filter

import (
	"io"

	"leveldb/hash"
)

func bloomHash(key []byte) uint32 {
	return hash.Hash(key, 0xbc9f1d34)
}

// BloomFilter filter represent a bloom filter.
type BloomFilter struct {
	bitsPerKey, k uint32
}

// NewBloomFilter create new initialized bloom filter for given
// bitsPerKey.
func NewBloomFilter(bitsPerKey int) *BloomFilter {
	// We intentionally round down to reduce probing cost a little bit
	k := uint32(bitsPerKey) * 69 / 100 // 0.69 =~ ln(2)
	if k < 1 {
		k = 1
	} else if k > 30 {
		k = 30
	}
	return &BloomFilter{uint32(bitsPerKey), k}
}

// Name return the name of this filter. i.e. "leveldb.BuiltinBloomFilter".
func (*BloomFilter) Name() string {
	return "leveldb.BuiltinBloomFilter"
}

// CreateFilter generate filter for given set of keys and write it to
// given buffer.
func (p *BloomFilter) CreateFilter(keys [][]byte, buf io.Writer) {
	// Compute bloom filter size (in both bits and bytes)
	bits := uint32(len(keys)) * p.bitsPerKey

	// For small n, we can see a very high false positive rate.  Fix it
	// by enforcing a minimum bloom filter length.
	if bits < 64 {
		bits = 64
	}

	bytes := (bits + 7) / 8
	bits = bytes * 8

	array := make([]byte, bytes)

	for _, key := range keys {
		// Use double-hashing to generate a sequence of hash values.
		// See analysis in [Kirsch,Mitzenmacher 2006].
		h := bloomHash(key)
		delta := (h >> 17) | (h << 15) // Rotate right 17 bits
		for i := uint32(0); i < p.k; i++ {
			bitpos := h % bits
			array[bitpos/8] |= (1 << (bitpos % 8))
			h += delta
		}
	}

	buf.Write(array)
	buf.Write([]byte{byte(p.k)})
}

// KeyMayMatch test whether given key on the list.
func (p *BloomFilter) KeyMayMatch(key, filter []byte) bool {
	l := uint32(len(filter))
	if l < 2 {
		return false
	}

	bits := (l - 1) * 8

	// Use the encoded k so that we can read filters generated by
	// bloom filters created using different parameters.
	k := uint32(filter[l-1])
	if k > 30 {
		// Reserved for potentially new encodings for short bloom filters.
		// Consider it a match.
		return true
	}

	h := bloomHash(key)
	delta := (h >> 17) | (h << 15) // Rotate right 17 bits
	for i := uint32(0); i < k; i++ {
		bitpos := h % bits
		if (uint32(filter[bitpos/8]) & (1 << (bitpos % 8))) == 0 {
			return false
		}
		h += delta
	}

	return true
}
