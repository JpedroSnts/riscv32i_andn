package simulador

import (
	"strings"
	"testing"

	"riscv/barramento"
	"riscv/programas"
	"riscv/video"
)

// TestProgramaAlfabeto roda o programa completo e confere o estado final.
func TestProgramaAlfabeto(t *testing.T) {
	sim, err := Novo(programas.Alfabeto)
	if err != nil {
		t.Fatalf("erro ao montar: %v", err)
	}

	passos, err := sim.Executar(10000, 0, nil)
	if err != nil {
		t.Fatalf("erro na execução: %v", err)
	}

	// 8 instruções de preparação + 8 voltas de 6 instruções + o ebreak
	if passos != 57 {
		t.Errorf("instruções executadas = %d, esperado 57", passos)
	}
	if !sim.CPU.Parou {
		t.Error("a CPU deveria ter parado no ebreak")
	}
	if sim.CPU.PC != 0x38 {
		t.Errorf("PC = %#x, esperado 0x38 (o ebreak)", sim.CPU.PC)
	}

	// registradores no final
	registradores := []struct {
		nome     string
		numero   int
		esperado uint32
	}{
		{"x4 (endereço na RAM)", 4, 256},
		{"x6 (ponteiro da VRAM, 8 posições à frente)", 6, 0x80020},
		{"x7 (a próxima letra seria 'i')", 7, 'i'},
		{"x8 (contador do laço)", 8, 0},
		{"x9 (última letra, já maiúscula)", 9, 'H'},
		{"x10 (máscara do ANDN)", 10, 32},
	}
	for _, r := range registradores {
		if obtido := sim.CPU.Regs[r.numero]; obtido != r.esperado {
			t.Errorf("%s = %#x, esperado %#x", r.nome, obtido, r.esperado)
		}
	}

	// o sw guardou o 'a' na RAM
	if valor, _ := sim.Mapa.Read32(256); valor != 'a' {
		t.Errorf("MEM[256] = %d, esperado %d ('a')", valor, 'a')
	}

	// a VRAM tem ABCDEFGH e a posição seguinte continua vazia
	for k := 0; k < 8; k++ {
		endereco := uint32(0x80000 + k*4)
		if valor, _ := sim.Mapa.Read32(endereco); valor != uint32('A'+k) {
			t.Errorf("VRAM[%#x] = %d, esperado %d", endereco, valor, 'A'+k)
		}
	}
	if valor, _ := sim.Mapa.Read32(0x80020); valor != 0 {
		t.Errorf("VRAM[0x80020] = %d, deveria estar vazia", valor)
	}

	// o que aparece no terminal
	tela, err := video.Montar(sim.Mapa)
	if err != nil {
		t.Fatalf("erro ao montar a tela: %v", err)
	}
	esperado := "|ABCDEFGH" + strings.Repeat(" ", video.Colunas-8) + "|"
	if linhas := strings.Split(tela, "\n"); linhas[1] != esperado {
		t.Errorf("primeira linha da tela = %q, esperado %q", linhas[1], esperado)
	}

	// no fim, o barramento está livre
	if sim.Barramento.Controle != barramento.Ocioso {
		t.Errorf("Controle = %v, esperado ocioso", sim.Barramento.Controle)
	}
}

// TestIntervaloDaTela confere que a função de tela é chamada a cada n instruções.
func TestIntervaloDaTela(t *testing.T) {
	sim, err := Novo(programas.Alfabeto)
	if err != nil {
		t.Fatalf("erro ao montar: %v", err)
	}

	var chamadas []int
	_, err = sim.Executar(10000, 20, func(passos int) {
		chamadas = append(chamadas, passos)
	})
	if err != nil {
		t.Fatalf("erro na execução: %v", err)
	}

	// o programa tem 57 instruções: chamadas nas de número 20 e 40
	if len(chamadas) != 2 || chamadas[0] != 20 || chamadas[1] != 40 {
		t.Errorf("chamadas = %v, esperado [20 40]", chamadas)
	}
}

// TestLimiteDePassos confere que um laço infinito é interrompido com erro.
func TestLimiteDePassos(t *testing.T) {
	// beq x0, x0, 0 desvia para ele mesmo para sempre
	sim, err := Novo([]uint32{0x00000063})
	if err != nil {
		t.Fatalf("erro ao montar: %v", err)
	}

	passos, err := sim.Executar(100, 0, nil)
	if err == nil {
		t.Error("deveria dar erro de limite de passos")
	}
	if passos != 100 {
		t.Errorf("passos = %d, esperado 100", passos)
	}
	if sim.CPU.Parou {
		t.Error("a CPU não deveria ter parado")
	}
}


// TestProgramaInvalido confere que uma instrução inválida interrompe a execução.
func TestProgramaInvalido(t *testing.T) {
	sim, err := Novo([]uint32{0x00000000}) // opcode 0 não existe
	if err != nil {
		t.Fatalf("erro ao montar: %v", err)
	}

	passos, err := sim.Executar(100, 0, nil)
	if err == nil {
		t.Error("deveria dar erro de instrução inválida")
	}
	if passos != 0 {
		t.Errorf("passos = %d, esperado 0", passos)
	}
}

// TestProgramaInvadeRegiaoReservada confere que um programa tão grande que
// passa da RAM e da VRAM e chega na região reservada é recusado.
func TestProgramaInvadeRegiaoReservada(t *testing.T) {
	// 0x90000 é o início da região reservada; a palavra de índice 0x90000/4 cai nela
	grande := make([]uint32, 0x90000/4+1)
	if _, err := Novo(grande); err == nil {
		t.Error("deveria recusar um programa que chega na região reservada")
	}
}