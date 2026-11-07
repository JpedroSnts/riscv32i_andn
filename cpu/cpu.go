package cpu

import "riscv/barramento"

type CPU struct {
	Regs[32] uint32
	PC uint32
	Barramento barramento.Barramento
}

func New(b barramento.Barramento) * CPU {
	return &CPU{Barramento: b}
}

func (c * CPU) write(r, v uint32){
	if r != 0 {
		c.Regs[r] = v
	}
}

func (c * CPU) Step() {
	inst := c.Barramento.Read32(c.PC)
	nextPC := c.PC + 4 

	switch opcode(inst) {
	case 0x33:
		r:= decodeR(inst)
		op, ok := operacaoR(r.funct3, r.funct7)
		if ok {
			res := ula(op, c.Regs[r.rs1], c.Regs[r.rs2])
			c.write(r.rd, res)
		}
	case 0x13: // Tipo-I aritmético/lógico
		op, ok := operacaoR(funct3(inst), funct7(inst))
		if ok {
			res := ula(op, c.Regs[rs1(inst)], uint32(immI(inst)))
			c.write(rd(inst), res)
		}
	}

	c.PC = nextPC
}