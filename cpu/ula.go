package cpu

type operacao int

const (
	opAdd operacao = iota
	opSub
	opSll
	opSlt
	opSltu
	opXor
	opSrl
	opSra
	opOr
	opAnd
	opAndn
)

func ula (op operacao , a , b uint32) uint32{
	switch op{
	case opAdd:
		return a + b
	case opSub:
		return a - b
	case opSll:
		return a <<(b & 0x1F)
	case opSlt:
		if int32(a) < int32(b) {
			return 1
		}
		return 0
	case opSltu:
		if a < b {
			return 1
		}
		return 0
	case opXor:
		return a ^ b 
	case opSrl:
		return a >> (b & 0x1F)
	case opSra:
		return uint32(int32(a) >> (b & 0x1F))
	case opOr:
		return a | b
	case opAnd:
		return a & b
	case opAndn:
		return a &^ b
	}
	return 0
}
