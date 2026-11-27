package cpu

import (
	"fmt"
	"riscv/barramento"
)

// VetorInterrupcao é o endereço da rotina de tratamento, na RAM.
const VetorInterrupcao = 0x1000

type CPU struct {
	Regs       [32]uint32
	PC         uint32
	Barramento barramento.Barramento
	Parou      bool // vira true quando a CPU executa EBREAK (fim do programa)

	LinhaInterrupcao barramento.FonteInterrupcao // se for nil, a CPU nunca é interrompida
	PCSalvo          uint32                      // onde o programa foi interrompido
	EmTratamento     bool                        // true enquanto a rotina de tratamento roda
}

func New(b barramento.Barramento) *CPU {
	return &CPU{Barramento: b}
}

func (c *CPU) write(r, v uint32) {
	if r != 0 {
		c.Regs[r] = v
	}
}

// Step executa uma instrução. Devolve erro se a instrução for inválida ou não suportada; nesse caso o PC e o estado da CPU não são alterados.
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
	case 0x73: // instruções de sistema: EBREAK e MRET
		switch inst {
		case 0x00100073: // EBREAK: fim do programa
			c.Parou = true
			return nil // o PC não avança, fica apontando para o ebreak
		case 0x30200073: // MRET: volta da rotina de interrupção
			if !c.EmTratamento {
				return c.invalida(inst) // MRET fora de uma rotina é erro
			}
			c.EmTratamento = false
			nextPC = c.PCSalvo
		default: // ecall e outras não são suportadas
			return c.invalida(inst)
		} // fecha o switch inst

	default: // opcode que não implementamos
		return c.invalida(inst)
	} // fecha o switch opcode(inst)

	// Depois de executar, verifica se há interrupção (figura 7 do enunciado). Não interrompe se já está numa rotina: interrupções não aninham.
	if !c.EmTratamento && c.LinhaInterrupcao != nil && c.LinhaInterrupcao.Interrupcao() {
		c.PCSalvo = nextPC // o programa continuará daqui depois do MRET
		c.EmTratamento = true
		nextPC = VetorInterrupcao
	}

	c.PC = nextPC
	return nil
}

// invalida monta o erro para uma instrução que a CPU não sabe executar.
func (c *CPU) invalida(inst uint32) error {
	return fmt.Errorf("instrução inválida %#08x no PC=%#x", inst, c.PC)
}
