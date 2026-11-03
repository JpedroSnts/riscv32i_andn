package cpu


func opcode (i uint32) uint32 { return i & 0x7F}
func rd (i uint32) uint32 {return (i >> 7) & 0x1F}
func funct3 (i uint32) uint32 {return (i >> 12) & 0x7}
func rs1 (i uint32) uint32 {return (i >> 15) & 0x1F}
func rs2 (i uint32) uint32 {return (i >> 20) & 0x1F}
func funct7 (i uint32) uint32 {return (i >> 25)}


func immI(i uint32) int32 {
	return int32(i) >> 20
}

func immS(i uint32) int32 {
	return (int32(i) >>25)<<5 | int32((i>>7) &0x1F)
}

func immB (i uint32) int32 {
	return (int32 (i)>> 31 <<12)|
	int32 ((i>>25)&0x3F)<<5 |
	int32((i>>8)&0xF)<<1
}

type instrR struct {
	rd, rs1, rs2 uint32
	funct3 		 uint32
	funct7		 uint32
}

func decodeR (i uint32) instrR {
	return instrR {
		rd: 	rd(i),
		rs1:	rs1(i),
		rs2:	rs2(i),
		funct3:	funct3(i),
		funct7:	funct7(i),	 
	}
}

func operacaoR(f3, f7 uint32) (operacao, bool) {
	switch f3 {
	case 0x0:
		if f7 == 0x20 {
			return opSub, true
		}
		if f7 == 0x00 {
			return opAdd, true
		}
	case 0x1:
		if f7 == 0x00 {
			return opSll, true
		}
	case 0x2:
		if f7 == 0x00 {
			return opSlt, true
		}
	case 0x3:
		if f7 == 0x00 {
			return opSltu, true
		}
	case 0x4:
		if f7 == 0x00 {
			return opXor, true
		}
	case 0x5:
		if f7 == 0x20 {
			return opSra, true
		}
		if f7 == 0x00 {
			return opSrl, true
		}
	case 0x6:
		if f7 == 0x00 {
			return opOr, true
		}
	case 0x7:
		if f7 == 0x20 {
			return opAndn, true
		}
		if f7 == 0x00 {
			return opAnd, true
		}
	}
	return 0, false
}