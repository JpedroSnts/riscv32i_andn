package simulador

import (
	"strings"
	"testing"

	"riscv/teclado"
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

// TestProgramaInvadeRegiaoReservada confere que um programa tão grande que passa da RAM e da VRAM e chega na região reservada é recusado.
func TestProgramaInvadeRegiaoReservada(t *testing.T) {
	// 0x90000 é o início da região reservada; a palavra de índice 0x90000/4 cai nela
	grande := make([]uint32, 0x90000/4+1)
	if _, err := Novo(grande); err == nil {
		t.Error("deveria recusar um programa que chega na região reservada")
	}
}

// TestTecladoNoMapa confere que o teclado responde nos endereços de periféricos
// e que a linha de interrupção do barramento acompanha o teclado.
func TestTecladoNoMapa(t *testing.T) {
	sim, err := Novo([]uint32{0x00100073})
	if err != nil {
		t.Fatalf("erro ao montar: %v", err)
	}

	// as interrupções vêm desligadas: apertar a tecla não interrompe
	sim.Teclado.Pressionar('i')
	if sim.Barramento.Interrupcao() {
		t.Error("com as interrupções desligadas, não deveria haver interrupção")
	}

	// o programa liga as interrupções escrevendo no CONTROLE, pelo mapa
	if err := sim.Mapa.Write32(teclado.RegControle, 1); err != nil {
		t.Fatalf("erro ao escrever no CONTROLE: %v", err)
	}
	if !sim.Barramento.Interrupcao() {
		t.Error("habilitado e com tecla pendente: deveria haver interrupção")
	}

	// o STATUS mostra a pendência e a leitura da TECLA a limpa
	if valor, _ := sim.Mapa.Read32(teclado.RegStatus); valor != 1 {
		t.Errorf("STATUS = %d, esperado 1", valor)
	}
	if valor, _ := sim.Mapa.Read32(teclado.RegTecla); valor != 'i' {
		t.Errorf("TECLA = %d, esperado %d ('i')", valor, 'i')
	}
	if sim.Barramento.Interrupcao() {
		t.Error("depois de ler a TECLA, a interrupção deveria cair")
	}
}

// TestTecladoNaoPassaPelaCache confere que a cache não guarda o teclado: o valor muda sozinho, e uma cache devolveria o antigo.
func TestTecladoNaoPassaPelaCache(t *testing.T) {
	sim, err := Novo([]uint32{0x00100073})
	if err != nil {
		t.Fatalf("erro ao montar: %v", err)
	}

	antes, _ := sim.Cache.Read32(teclado.RegStatus)
	sim.Teclado.Pressionar('i')
	depois, _ := sim.Cache.Read32(teclado.RegStatus)

	if antes != 0 || depois != 1 {
		t.Errorf("STATUS antes/depois = %d/%d, esperado 0/1", antes, depois)
	}
	if sim.Cache.Hits != 0 || sim.Cache.Misses != 0 {
		t.Errorf("hits/misses = %d/%d, o teclado não deveria contar na cache",
			sim.Cache.Hits, sim.Cache.Misses)
	}

	// escrever no CONTROLE pela cache também chega ao teclado
	if err := sim.Cache.Write32(teclado.RegControle, 1); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !sim.Barramento.Interrupcao() {
		t.Error("a escrita no CONTROLE pela cache deveria ter ligado as interrupções")
	}
}