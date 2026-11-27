package simulador

import (
	"strings"
	"testing"

	"riscv/barramento"
	"riscv/cpu"
	"riscv/programas"
	"riscv/teclado"
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

// ---------- interrupção por teclado ----------

const (
	passosSemInterrupcao = 414 // instruções do programa principal, sem rotina
	passosDaRotina       = 20  // instruções da rotina de tratamento
	enderecoContador     = 0x740
	enderecoMarcaF       = 0x80000 + 160 // linha 1 da tela
)

// novoComRotina monta o simulador com o programa principal em 0x0000
// e a rotina de tratamento em 0x1000.
func novoComRotina(t *testing.T) *Simulador {
	t.Helper()
	sim, err := Novo(programas.InterrupcaoPrincipal)
	if err != nil {
		t.Fatalf("erro ao montar: %v", err)
	}
	if err := sim.Mapa.Load(cpu.VetorInterrupcao, programas.InterrupcaoRotina); err != nil {
		t.Fatalf("erro ao carregar a rotina: %v", err)
	}
	return sim
}

// executarComTeclas roda o programa e aperta teclas em pontos exatos:
// teclas[n] é apertada depois de n instruções executadas.
func executarComTeclas(t *testing.T, sim *Simulador, teclas map[int]byte) int {
	t.Helper()
	passos, err := sim.Executar(10000, 1, func(passos int) {
		if tecla, ok := teclas[passos]; ok {
			sim.Teclado.Pressionar(tecla)
		}
	})
	if err != nil {
		t.Fatalf("erro na execução: %v", err)
	}
	return passos
}

// conferirDemonstracao confere o estado final para um número de interrupções.
func conferirDemonstracao(t *testing.T, sim *Simulador, passos, interrupcoes int) {
	t.Helper()

	esperado := passosSemInterrupcao + interrupcoes*passosDaRotina
	if passos != esperado {
		t.Errorf("instruções executadas = %d, esperado %d", passos, esperado)
	}
	if !sim.CPU.Parou || sim.CPU.EmTratamento {
		t.Errorf("Parou = %v, EmTratamento = %v", sim.CPU.Parou, sim.CPU.EmTratamento)
	}

	// o programa principal terminou o trabalho, apesar das interrupções
	if sim.CPU.Regs[8] != 200 {
		t.Errorf("x8 = %d, esperado 200 (o laço terminou)", sim.CPU.Regs[8])
	}
	if valor, _ := sim.Mapa.Read32(enderecoMarcaF); valor != 'F' {
		t.Errorf("a linha 1 da tela tem %d, esperado %d ('F')", valor, 'F')
	}

	// a rotina restaurou os registradores
	for numero, esperado := range map[int]uint32{28: 77, 29: 88, 30: 99} {
		if obtido := sim.CPU.Regs[numero]; obtido != esperado {
			t.Errorf("x%d = %d, esperado %d (a rotina não restaurou)", numero, obtido, esperado)
		}
	}

	// o contador e a tela mostram quantas vezes a rotina rodou
	if valor, _ := sim.Mapa.Read32(enderecoContador); valor != uint32(interrupcoes) {
		t.Errorf("contador = %d, esperado %d", valor, interrupcoes)
	}
	for k := 0; k <= interrupcoes+1; k++ {
		endereco := uint32(0x80000 + k*4)
		valor, _ := sim.Mapa.Read32(endereco)
		if k < interrupcoes && valor != '*' {
			t.Errorf("VRAM[%#x] = %d, esperado %d ('*')", endereco, valor, '*')
		}
		if k >= interrupcoes && valor != 0 {
			t.Errorf("VRAM[%#x] = %d, deveria estar vazia", endereco, valor)
		}
	}
}

// TestInterrupcaoSemTecla confere o programa sem nenhuma tecla.
func TestInterrupcaoSemTecla(t *testing.T) {
	sim := novoComRotina(t)
	passos := executarComTeclas(t, sim, nil)
	conferirDemonstracao(t, sim, passos, 0)
}

// TestInterrupcaoTeclaIgnorada confere que outras teclas não interrompem.
func TestInterrupcaoTeclaIgnorada(t *testing.T) {
	sim := novoComRotina(t)
	passos := executarComTeclas(t, sim, map[int]byte{50: 'a', 100: 'I', 150: '\n'})
	conferirDemonstracao(t, sim, passos, 0)
}

// TestInterrupcaoUmaTecla confere uma interrupção no meio do laço.
func TestInterrupcaoUmaTecla(t *testing.T) {
	sim := novoComRotina(t)
	passos := executarComTeclas(t, sim, map[int]byte{50: 'i'})
	conferirDemonstracao(t, sim, passos, 1)
}

// TestInterrupcaoDuasTeclas confere duas interrupções em momentos diferentes.
func TestInterrupcaoDuasTeclas(t *testing.T) {
	sim := novoComRotina(t)
	passos := executarComTeclas(t, sim, map[int]byte{50: 'i', 200: 'i'})
	conferirDemonstracao(t, sim, passos, 2)
}

// TestInterrupcaoTeclaAntesDeHabilitar confere que uma tecla apertada antes
// de o programa ligar as interrupções fica pendente e é tratada logo depois.
func TestInterrupcaoTeclaAntesDeHabilitar(t *testing.T) {
	sim := novoComRotina(t)
	sim.Teclado.Pressionar('i') // antes da primeira instrução
	passos := executarComTeclas(t, sim, nil)
	conferirDemonstracao(t, sim, passos, 1)
}

// TestInterrupcaoTeclaDuranteARotinaAntesDeLer confere que uma tecla apertada
// enquanto a anterior ainda não foi lida é a mesma pendência.
func TestInterrupcaoTeclaDuranteARotinaAntesDeLer(t *testing.T) {
	sim := novoComRotina(t)
	// a rotina começa depois da instrução 51 e lê a TECLA na instrução 58
	passos := executarComTeclas(t, sim, map[int]byte{50: 'i', 54: 'i'})
	conferirDemonstracao(t, sim, passos, 1)
}

// TestInterrupcaoTeclaDuranteARotinaDepoisDeLer confere que uma tecla apertada
// depois de a rotina ler a anterior é uma nova interrupção, tratada depois do MRET.
func TestInterrupcaoTeclaDuranteARotinaDepoisDeLer(t *testing.T) {
	sim := novoComRotina(t)
	passos := executarComTeclas(t, sim, map[int]byte{50: 'i', 60: 'i'})
	conferirDemonstracao(t, sim, passos, 2)
}

// TestInterrupcaoRotinaNaoMudaOQueNaoUsa confere que registradores que a rotina
// não usa chegam ao fim com o valor que o programa principal deu.
func TestInterrupcaoRotinaNaoMudaOQueNaoUsa(t *testing.T) {
	sim := novoComRotina(t)
	executarComTeclas(t, sim, map[int]byte{50: 'i'})

	// x9 = limite do laço, x6 = VRAM, x10 = teclado, x11 = 1
	esperados := map[int]uint32{9: 200, 6: 0x80000, 10: 0x9FC00, 11: 1}
	for numero, esperado := range esperados {
		if obtido := sim.CPU.Regs[numero]; obtido != esperado {
			t.Errorf("x%d = %#x, esperado %#x", numero, obtido, esperado)
		}
	}
}
