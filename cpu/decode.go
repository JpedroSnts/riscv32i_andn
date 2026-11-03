package cpu

type Decodificador struct{}

func opcode (i uint32) uint32 { return i & 0x7F}
func rd (i uint32) uint32 {return (i >> 7) & 0x1F}
func funct3 (i uint32) uint32 {return (i >> 12) & 0x7}
func rs1 (i uint32) uint32 {return (i >> 15) & 0x1F}
func rs2 (i uint32) uint32 {return (i >> 20) & 0x1F}
func funct7 (i uint32) uint32 {return (i >> 25)}


func immI(i uint32) int32 {
	return int32(i) >> 20
}