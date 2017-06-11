package homeworlds

type Bank struct {
	bits uint32
}

func (b Bank) IsEmpty() bool {
	return b.bits == 0
}

func (b Bank) Has(p Piece) bool {
	return (b.bits>>(p*2))&0x3 != 0
}

func (b Bank) Get(p Piece) int {
	return int((b.bits >> (p * 2)) & 0x3)
}

func (b *Bank) Set(p Piece, n int) {
	b.bits &^= 3 << (p * 2)
	b.bits |= (uint32(n) & 3) << (p * 2)
}

func (b *Bank) Take(p Piece) {
	n := b.Get(p)
	if n != 0 {
		b.Set(p, n-1)
	}
}

func (b *Bank) Put(p Piece) {
	n := b.Get(p)
	if n < 3 {
		b.Set(p, n+1)
	}
}

func (b Bank) Largest() Size {
	x := b.bits
	x |= x >> 12
	x |= x >> 6

	i := 0
	for ; x != 0; x >>= 2 {
		i++
	}
	return Size(i)
}

func (b Bank) SmallestOfColor(c Color) Size {
	x := b.bits >> (c * 6) & 63
	i := 1
	for ; x&3 != 0; x >>= 2 {
		i++
	}
	return Size(i)
}

func (b Bank) LargestOfColor(c Color) Size {
	x := b.bits >> (c * 6) & 63
	i := 0
	for ; x&3 != 0; x >>= 2 {
		i++
	}
	return Size(i)
}

func (b Bank) HasColor(c Color) bool {
	return b.bits>>(c*6)&63 != 0
}

type BankIterator struct {
	i    int
	bits uint32
}

func (bi BankIterator) Done() bool {
	return bi.bits == 0
}

func (bi *BankIterator) Next() {
	bi.i++
	bi.bits >>= 2
}

func (bi BankIterator) Piece() Piece {
	return Piece(bi.i)
}

func (bi BankIterator) Count() int {
	return int(bi.bits & 3)
}

func (b Bank) Iter() BankIterator {
	return BankIterator{i: 0, bits: b.bits}
}
