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
		op, ok := operacaoI(funct3(inst), funct7(inst))
		if ok {
			res := ula(op, c.Regs[rs1(inst)], uint32(immI(inst)))
			c.write(rd(inst), res)
		}
	case 0x03: // LW (Tipo-I): rd = MEM[rs1 + imm]
		if funct3(inst) == 0x2 {
			addr:= c.Regs[rs1(inst)] + uint32(immI(inst))
			c.write(rd(inst), c.Barramento.Read32(addr))
		}
	case 0x23: // SW (Tipo-S): MEM[rs1 + imm] = rs2
		if funct3(inst) == 0x2 {
			addr := c.Regs[rs1(inst)] + uint32(immS(inst))
			c.Barramento.Write32(addr, c.Regs[(inst)])
		}
	case 0x63: // Branches (Tipo-B)
		a, b := c.Regs[rs1(inst)], c.Regs[rs2(inst)]

		// desvio vale true se a condição do branch for verdadeira
		desvio := false

		switch funct3(inst) {
		case 0x0: // BEQ: desvia se igual
			desvio = a == b
		case 0x1: //BNE: desvia se diferente
			desvio = a != b
		case 0x4: // BLT: desvia se menor, com sinal
			desvio = int32(a) < int32(b)
		case 0x5: // BGE: desvia se maior ou igual, com sinal
			desvio = int32(a) >= int32(b)
		case 0x6: // BLTU: desvia se menor, sem sinal
			desvio = a < b
		case 0x7: // BGEU: desvia se maior ou igual, sem sinal
			desvio = a >= b
		}
		
		// Se houver desvio, o PC pula para PC + imediato.
		// Senão, nextPC continua valendo PC + 4 (próxima instrução).
		if desvio {
			nextPC = c.PC + uint32(immB(inst))
		}
	}

	c.PC = nextPC
}