package cpu

import (
	"fmt"
	"riscv/barramento"
)

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

// Step executa uma instrução. Devolve erro se a instrução for inválida
// ou não suportada; nesse caso o PC e o estado da CPU não são alterados.
func (c *CPU) Step() error {
	inst, err := c.Barramento.Read32(c.PC)
	if err != nil {
		return err // não foi possível buscar a instrução
	}
	nextPC := c.PC + 4

	switch opcode(inst) {
	case 0x33: // Tipo-R
		r := decodeR(inst)
		op, ok := operacaoR(r.funct3, r.funct7)
		if !ok {
			return c.invalida(inst)
		}
		res := ula(op, c.Regs[r.rs1], c.Regs[r.rs2])
		c.write(r.rd, res)

	case 0x13: // Tipo-I aritmético/lógico
		op, ok := operacaoI(funct3(inst), funct7(inst))
		if !ok {
			return c.invalida(inst)
		}
		res := ula(op, c.Regs[rs1(inst)], uint32(immI(inst)))
		c.write(rd(inst), res)

	case 0x03: // LW
		if funct3(inst) != 0x2 {
			return c.invalida(inst)
		}
		addr := c.Regs[rs1(inst)] + uint32(immI(inst))
		dado, err := c.Barramento.Read32(addr)
		if err != nil {
			return err
		}
		c.write(rd(inst), dado)

	case 0x23: // SW
		if funct3(inst) != 0x2 {
			return c.invalida(inst)
		}
		addr := c.Regs[rs1(inst)] + uint32(immS(inst))
		if err := c.Barramento.Write32(addr, c.Regs[rs2(inst)]); err != nil {
			return err
		}

	case 0x63: // Branches
		a, b := c.Regs[rs1(inst)], c.Regs[rs2(inst)]
		desvio := false
		switch funct3(inst) {
		case 0x0: // BEQ
			desvio = a == b
		case 0x1: // BNE
			desvio = a != b
		case 0x4: // BLT
			desvio = int32(a) < int32(b)
		case 0x5: // BGE
			desvio = int32(a) >= int32(b)
		case 0x6: // BLTU
			desvio = a < b
		case 0x7: // BGEU
			desvio = a >= b
		default:
			return c.invalida(inst)
		}
		if desvio {
			nextPC = c.PC + uint32(immB(inst))
		}

	default: // opcode que não implementamos
		return c.invalida(inst)
	}

	c.PC = nextPC
	return nil
}

// invalida monta o erro para uma instrução que a CPU não sabe executar.
func (c *CPU) invalida(inst uint32) error {
	return fmt.Errorf("instrução inválida %#08x no PC=%#x", inst, c.PC)
}