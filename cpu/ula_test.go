package cpu

import "testing"

// TestULA confere cada operação com valores conhecidos.
func TestULA(t *testing.T) {
	casos := []struct {
		nome     string
		op       operacao
		a, b     uint32
		esperado uint32
	}{
		// soma e subtração (o resultado "dá a volta" em 32 bits)
		{"add 5+3", opAdd, 5, 3, 8},
		{"add com overflow", opAdd, 0xFFFFFFFF, 1, 0},
		{"sub 5-3", opSub, 5, 3, 2},
		{"sub abaixo de zero", opSub, 0, 1, 0xFFFFFFFF},

		// lógicas, com 1100 e 1010
		{"and", opAnd, 0b1100, 0b1010, 0b1000},
		{"or", opOr, 0b1100, 0b1010, 0b1110},
		{"xor", opXor, 0b1100, 0b1010, 0b0110},

		// ANDN: a ULA deve dar um resultado DIFERENTE do AND
		{"andn (diferente do and)", opAndn, 0b1100, 0b1010, 0b0100},
		{"andn: minúscula vira maiúscula", opAndn, 'a', 0x20, 'A'},
		{"andn com b zero não muda a", opAndn, 0xABCD, 0, 0xABCD},
		{"andn com b todo 1 zera tudo", opAndn, 0xABCD, 0xFFFFFFFF, 0},

		// deslocamentos
		{"sll 1<<31", opSll, 1, 31, 0x80000000},
		{"sll por 0", opSll, 5, 0, 5},
		{"sll por 32 usa só 5 bits (= 0)", opSll, 5, 32, 5},
		{"srl entra zero", opSrl, 0xF0000000, 4, 0x0F000000},
		{"srl 31", opSrl, 0x80000000, 31, 1},
		{"sra copia o sinal", opSra, 0xF0000000, 4, 0xFF000000},
		{"sra 31 de negativo", opSra, 0x80000000, 31, 0xFFFFFFFF},
		{"sra de positivo", opSra, 0x40000000, 30, 1},

		// comparações: com sinal e sem sinal dão respostas diferentes
		{"slt -1 < 0", opSlt, 0xFFFFFFFF, 0, 1},
		{"slt 0 < -1", opSlt, 0, 0xFFFFFFFF, 0},
		{"slt igual", opSlt, 7, 7, 0},
		{"sltu 0xFFFFFFFF < 0", opSltu, 0xFFFFFFFF, 0, 0},
		{"sltu 0 < 0xFFFFFFFF", opSltu, 0, 0xFFFFFFFF, 1},
	}

	for _, c := range casos {
		if obtido := ula(c.op, c.a, c.b); obtido != c.esperado {
			t.Errorf("%s: obtido %#x, esperado %#x", c.nome, obtido, c.esperado)
		}
	}
}

// TestOperacaoR confere a tradução de funct3/funct7 do Tipo-R.
func TestOperacaoR(t *testing.T) {
	casos := []struct {
		f3, f7 uint32
		op operacao
		valido bool
	}{
		{0x0, 0x00, opAdd, true},
		{0x0, 0x20, opSub, true},
		{0x1, 0x00, opSll, true},
		{0x2, 0x00, opSlt, true},
		{0x3, 0x00, opSltu, true},
		{0x4, 0x00, opXor, true},
		{0x5, 0x00, opSrl, true},
		{0x5, 0x20, opSra, true},
		{0x6, 0x00, opOr, true},
		{0x7, 0x00, opAnd, true},
		{0x7, 0x20, opAndn, true}, // a instrução do grupo 12

		// combinações que não existem
		{0x0, 0x01, 0, false},
		{0x1, 0x20, 0, false},
		{0x7, 0x40, 0, false},
	}
	for _, c := range casos {
		op, ok := operacaoR(c.f3, c.f7)
		if ok != c.valido {
			t.Errorf("operacaoR(%#x, %#x): ok = %v, esperado %v", c.f3, c.f7, ok, c.valido)
			continue
		}
		if ok && op != c.op {
			t.Errorf("operacaoR(%#x, %#x) = %d, esperado %d", c.f3, c.f7, op, c.op)
		}
	}
}

// TestOperacaoI confere a tradução de funct3/funct7 do Tipo-I.
func TestOperacaoI(t *testing.T) {
	casos := []struct {
		f3, f7 uint32
		op     operacao
		valido bool
	}{
		{0x0, 0x00, opAdd, true},  // ADDI
		{0x0, 0x7F, opAdd, true},  // ADDI ignora o funct7 (são bits do imediato)
		{0x1, 0x00, opSll, true},  // SLLI
		{0x2, 0x00, opSlt, true},  // SLTI
		{0x3, 0x00, opSltu, true}, // SLTIU
		{0x4, 0x00, opXor, true},  // XORI
		{0x5, 0x00, opSrl, true},  // SRLI
		{0x5, 0x20, opSra, true},  // SRAI
		{0x6, 0x00, opOr, true},   // ORI
		{0x7, 0x00, opAnd, true},  // ANDI
		{0x7, 0x20, opAnd, true},  // não existe ANDN com imediato: continua ANDI

		// combinações que não existem
		{0x1, 0x20, 0, false}, // SLLI com funct7 errado
		{0x5, 0x01, 0, false}, // shift à direita com funct7 errado
	}

	for _, c := range casos {
		op, ok := operacaoI(c.f3, c.f7)
		if ok != c.valido {
			t.Errorf("operacaoI(%#x, %#x): ok = %v, esperado %v", c.f3, c.f7, ok, c.valido)
			continue
		}
		if ok && op != c.op {
			t.Errorf("operacaoI(%#x, %#x) = %d, esperado %d", c.f3, c.f7, op, c.op)
		}
	}
}