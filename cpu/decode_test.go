package cpu

import "testing"

// TestCamposANDN confere o recorte dos campos com um andn x3, x1, x2.
func TestCamposANDN(t *testing.T) {
	instrucao := uint32(0x4020F1B3) // andn x3, x1, x2

	// Cada caso: nome do campo, o que a função devolveu e o que se esperava.
	casos := []struct {
		nome     string
		obtido   uint32
		esperado uint32
	}{
		{"opcode", opcode(instrucao), 0x33}, // 0110011 = Tipo-R
		{"rd", rd(instrucao), 3},
		{"funct3", funct3(instrucao), 0x7}, // 111, igual ao do AND
		{"rs1", rs1(instrucao), 1},
		{"rs2", rs2(instrucao), 2},
		{"funct7", funct7(instrucao), 0x20}, // 0100000, o que diferencia do AND
	}

	for _, c := range casos {
		if c.obtido != c.esperado {
			t.Errorf("%s = %#x, esperado %#x", c.nome, c.obtido, c.esperado)
		}
	}
}

// TestImmI confere o imediato do Tipo-I (ADDI, LW...), com positivos e negativos.
func TestImmI(t *testing.T) {
	casos := []struct {
		nome      string
		instrucao uint32
		esperado  int32
	}{
		{"addi x1, x0, 12", 0x00C00093, 12},
		{"addi x1, x0, -1", 0xFFF00093, -1},
		{"addi x8, x8, -1", 0xFFF40413, -1},
		{"addi x1, x0, 2047 (maior)", 0x7FF00093, 2047},
		{"addi x1, x0, -2048 (menor)", 0x80000093, -2048},
	}

	for _, c := range casos {
		if obtido := immI(c.instrucao); obtido != c.esperado {
			t.Errorf("%s: immI = %d, esperado %d", c.nome, obtido, c.esperado)
		}
	}
}

// TestImmS confere o imediato do Tipo-S (SW), que vem partido em dois pedaços.
func TestImmS(t *testing.T) {
	casos := []struct {
		nome      string
		instrucao uint32
		esperado  int32
	}{
		{"sw x3, 0(x4)", 0x00322023, 0},
		{"sw x7, 4(x6)", 0x00732223, 4},
		{"sw x2, -4(x1)", 0xFE20AE23, -4},
		{"sw x2, 2047(x1) (maior)", 0x7E20AFA3, 2047},
	}

	for _, c := range casos {
		if obtido := immS(c.instrucao); obtido != c.esperado {
			t.Errorf("%s: immS = %d, esperado %d", c.nome, obtido, c.esperado)
		}
	}
}

// TestImmB confere o imediato do Tipo-B (branches), espalhado em 4 pedaços.
func TestImmB(t *testing.T) {
	casos := []struct {
		nome      string
		instrucao uint32
		esperado  int32
	}{
		{"beq +16", 0x00000863, 16},
		{"beq -8", 0xFE000CE3, -8},
		{"bne -20 (o do nosso laço)", 0xFE0416E3, -20},
		{"beq +2048 (usa o imm[11])", 0x000000E3, 2048},
		{"beq +4094 (maior)", 0x7E000FE3, 4094},
		{"beq -4096 (menor)", 0x80000063, -4096},
	}

	for _, c := range casos {
		if obtido := immB(c.instrucao); obtido != c.esperado {
			t.Errorf("%s: immB = %d, esperado %d", c.nome, obtido, c.esperado)
		}
	}
}
