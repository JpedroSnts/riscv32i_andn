package teclado

import "testing"

// ler lê um registrador e falha o teste se der erro.
func ler(t *testing.T, k *Teclado, endereco uint32) uint32 {
	t.Helper()
	valor, err := k.Read32(endereco)
	if err != nil {
		t.Fatalf("Read32(%#x): erro inesperado: %v", endereco, err)
	}
	return valor
}

// escrever escreve num registrador e falha o teste se der erro.
func escrever(t *testing.T, k *Teclado, endereco, valor uint32) {
	t.Helper()
	if err := k.Write32(endereco, valor); err != nil {
		t.Fatalf("Write32(%#x): erro inesperado: %v", endereco, err)
	}
}

// TestEstadoInicial confere que o teclado nasce quieto e desligado.
func TestEstadoInicial(t *testing.T) {
	k := Novo()

	if ler(t, k, RegStatus) != 0 {
		t.Error("STATUS deveria ser 0 (nenhuma tecla pendente)")
	}
	if ler(t, k, RegControle) != 0 {
		t.Error("CONTROLE deveria ser 0 (interrupções desligadas)")
	}
	if k.Interrupcao() {
		t.Error("não deveria haver interrupção")
	}
}

// TestPressionarMarcaPendencia confere que a tecla fica pendente mesmo desabilitado.
func TestPressionarMarcaPendencia(t *testing.T) {
	k := Novo()
	k.Pressionar('i')

	if ler(t, k, RegStatus) != 1 {
		t.Error("STATUS deveria ser 1 depois de apertar a tecla")
	}
	if k.Interrupcao() {
		t.Error("com as interrupções desligadas não deveria haver interrupção")
	}
}

// TestInterrupcaoSoComHabilitacao confere a linha de interrupção nos quatro casos.
func TestInterrupcaoSoComHabilitacao(t *testing.T) {
	k := Novo()

	// desligado e sem tecla
	if k.Interrupcao() {
		t.Error("desligado e sem tecla: não deveria haver interrupção")
	}

	// ligado e sem tecla
	escrever(t, k, RegControle, 1)
	if k.Interrupcao() {
		t.Error("ligado e sem tecla: não deveria haver interrupção")
	}

	// ligado e com tecla
	k.Pressionar('i')
	if !k.Interrupcao() {
		t.Error("ligado e com tecla: deveria haver interrupção")
	}

	// desligando com a tecla ainda pendente
	escrever(t, k, RegControle, 0)
	if k.Interrupcao() {
		t.Error("desligado com tecla pendente: não deveria haver interrupção")
	}
	if ler(t, k, RegStatus) != 1 {
		t.Error("a pendência deveria continuar marcada no STATUS")
	}
}

// TestOutrasTeclasSaoIgnoradas confere que só a tecla 'i' conta.
func TestOutrasTeclasSaoIgnoradas(t *testing.T) {
	k := Novo()
	escrever(t, k, RegControle, 1)

	for _, tecla := range []byte{'a', 'I', 'x', ' ', '\n', 0, 255} {
		k.Pressionar(tecla)
	}
	if ler(t, k, RegStatus) != 0 {
		t.Error("STATUS deveria continuar 0")
	}
	if k.Interrupcao() {
		t.Error("não deveria haver interrupção")
	}
}

// TestLerTeclaLimpaAPendencia confere o código da tecla e a limpeza ao ler.
func TestLerTeclaLimpaAPendencia(t *testing.T) {
	k := Novo()
	escrever(t, k, RegControle, 1)
	k.Pressionar('i')

	if tecla := ler(t, k, RegTecla); tecla != 'i' {
		t.Errorf("TECLA = %d, esperado %d ('i')", tecla, 'i')
	}
	if ler(t, k, RegStatus) != 0 {
		t.Error("STATUS deveria voltar a 0 depois de ler a TECLA")
	}
	if k.Interrupcao() {
		t.Error("a interrupção deveria ter sido atendida")
	}
}

// TestLerStatusNaoLimpa confere que só ler a TECLA limpa a pendência.
func TestLerStatusNaoLimpa(t *testing.T) {
	k := Novo()
	k.Pressionar('i')

	ler(t, k, RegStatus)
	ler(t, k, RegStatus)
	if ler(t, k, RegStatus) != 1 {
		t.Error("ler o STATUS não deveria limpar a pendência")
	}
}

// TestNovaTeclaDepoisDeLer confere que dá para apertar de novo depois de tratar.
func TestNovaTeclaDepoisDeLer(t *testing.T) {
	k := Novo()
	escrever(t, k, RegControle, 1)

	k.Pressionar('i')
	ler(t, k, RegTecla)
	if k.Interrupcao() {
		t.Fatal("depois de ler, não deveria haver interrupção")
	}

	k.Pressionar('i')
	if !k.Interrupcao() {
		t.Error("a segunda tecla deveria gerar outra interrupção")
	}
}

// TestTeclaRepetidaEnquantoPendente confere que continua uma só pendência.
func TestTeclaRepetidaEnquantoPendente(t *testing.T) {
	k := Novo()
	k.Pressionar('i')
	k.Pressionar('i')
	k.Pressionar('i')

	ler(t, k, RegTecla) // uma leitura basta para limpar
	if ler(t, k, RegStatus) != 0 {
		t.Error("uma leitura da TECLA deveria limpar a pendência")
	}
}

// TestControleSoOBit0 confere que só o bit 0 liga as interrupções.
func TestControleSoOBit0(t *testing.T) {
	casos := []struct {
		escrito  uint32
		esperado uint32
	}{
		{1, 1},
		{0, 0},
		{3, 1},          // bit 0 ligado, o resto é ignorado
		{0xFFFFFFFE, 0}, // bit 0 desligado
		{0xFFFFFFFF, 1},
	}

	for _, c := range casos {
		k := Novo()
		escrever(t, k, RegControle, c.escrito)
		if obtido := ler(t, k, RegControle); obtido != c.esperado {
			t.Errorf("escrito %#x: CONTROLE = %d, esperado %d", c.escrito, obtido, c.esperado)
		}
	}
}

// TestRegistradoresSomenteLeitura confere que STATUS e TECLA não aceitam escrita.
func TestRegistradoresSomenteLeitura(t *testing.T) {
	k := Novo()
	k.Pressionar('i')

	for _, endereco := range []uint32{RegStatus, RegTecla} {
		if err := k.Write32(endereco, 0); err == nil {
			t.Errorf("Write32(%#x): deveria dar erro (somente leitura)", endereco)
		}
	}
	// a tentativa de escrita não pode ter limpado a pendência
	if ler(t, k, RegStatus) != 1 {
		t.Error("a escrita recusada não deveria alterar a pendência")
	}
}

// TestEnderecosInvalidos confere o que não é registrador, nos dois sentidos.
func TestEnderecosInvalidos(t *testing.T) {
	enderecos := []uint32{
		RegControle + 4, // 0x9FC0C, logo depois do último registrador
		Base + Tamanho - 4,
		Base + 1, // desalinhado
		Base - 4, // antes da região
	}

	for _, endereco := range enderecos {
		k := Novo()
		if _, err := k.Read32(endereco); err == nil {
			t.Errorf("Read32(%#x): deveria dar erro", endereco)
		}
		if err := k.Write32(endereco, 1); err == nil {
			t.Errorf("Write32(%#x): deveria dar erro", endereco)
		}
	}
}

// TestEnderecosDosRegistradores confere os valores do mapa do teclado.
func TestEnderecosDosRegistradores(t *testing.T) {
	if RegStatus != 0x9FC00 || RegTecla != 0x9FC04 || RegControle != 0x9FC08 {
		t.Errorf("endereços = %#x, %#x, %#x; esperado 0x9FC00, 0x9FC04, 0x9FC08",
			RegStatus, RegTecla, RegControle)
	}
	if Base+Tamanho != 0xA0000 {
		t.Errorf("a região termina em %#x, esperado 0xA0000", Base+Tamanho)
	}
}