package barramento

import "testing"

// fonteFalsa é um dispositivo que pede interrupção quando pedindo é true.
type fonteFalsa struct {
	pedindo bool
}

func (f *fonteFalsa) Interrupcao() bool {
	return f.pedindo
}

// TestSemFontes confere que um barramento sem fontes nunca interrompe.
func TestSemFontes(t *testing.T) {
	s := NovoSistema(&destinoFalso{palavras: map[uint32]uint32{}})

	if s.Interrupcao() {
		t.Error("sem fontes, não deveria haver interrupção")
	}
}

// TestUmaFonte confere a linha com uma fonte, nos dois estados.
func TestUmaFonte(t *testing.T) {
	s := NovoSistema(&destinoFalso{palavras: map[uint32]uint32{}})
	f := &fonteFalsa{}
	s.AdicionarFonte(f)

	if s.Interrupcao() {
		t.Error("fonte quieta: não deveria haver interrupção")
	}
	f.pedindo = true
	if !s.Interrupcao() {
		t.Error("fonte pedindo: deveria haver interrupção")
	}
}

// TestLinhaAcompanhaAFonte confere que a linha é consultada na hora, e não guardada: se a fonte para de pedir, a linha cai.
func TestLinhaAcompanhaAFonte(t *testing.T) {
	s := NovoSistema(&destinoFalso{palavras: map[uint32]uint32{}})
	f := &fonteFalsa{pedindo: true}
	s.AdicionarFonte(f)

	if !s.Interrupcao() {
		t.Fatal("deveria haver interrupção")
	}
	f.pedindo = false
	if s.Interrupcao() {
		t.Error("a fonte parou de pedir: a linha deveria cair")
	}
	f.pedindo = true
	if !s.Interrupcao() {
		t.Error("a fonte voltou a pedir: a linha deveria subir de novo")
	}
}

// TestVariasFontes confere que basta uma fonte pedindo.
func TestVariasFontes(t *testing.T) {
	s := NovoSistema(&destinoFalso{palavras: map[uint32]uint32{}})
	a, b, c := &fonteFalsa{}, &fonteFalsa{}, &fonteFalsa{}
	s.AdicionarFonte(a)
	s.AdicionarFonte(b)
	s.AdicionarFonte(c)

	if s.Interrupcao() {
		t.Error("nenhuma pedindo: não deveria haver interrupção")
	}
	c.pedindo = true // só a última
	if !s.Interrupcao() {
		t.Error("a última pedindo: deveria haver interrupção")
	}
	a.pedindo = true // agora duas
	if !s.Interrupcao() {
		t.Error("duas pedindo: deveria haver interrupção")
	}
	a.pedindo, c.pedindo = false, false
	if s.Interrupcao() {
		t.Error("todas quietas: não deveria haver interrupção")
	}
}

// TestFonteNulaEhIgnorada confere que registrar uma fonte nula não quebra a linha.
func TestFonteNulaEhIgnorada(t *testing.T) {
	s := NovoSistema(&destinoFalso{palavras: map[uint32]uint32{}})
	s.AdicionarFonte(nil)

	if s.Interrupcao() {
		t.Error("não deveria haver interrupção")
	}
}

// TestConsultarNaoMexeNasLinhas confere que perguntar pela interrupção não altera as linhas de endereço, dados e controle.
func TestConsultarNaoMexeNasLinhas(t *testing.T) {
	destino := &destinoFalso{palavras: map[uint32]uint32{0x100: 42}}
	s := NovoSistema(destino)
	s.AdicionarFonte(&fonteFalsa{pedindo: true})

	s.Read32(0x100) // deixa as linhas com valores conhecidos
	endereco, dados, controle := s.Endereco, s.Dados, s.Controle

	s.Interrupcao()
	s.Interrupcao()

	if s.Endereco != endereco || s.Dados != dados || s.Controle != controle {
		t.Error("consultar a interrupção não deveria alterar as linhas do barramento")
	}
}

// TestInterrupcaoIndependeDosAcessos confere que a linha funciona mesmo quando a memória está falhando.
func TestInterrupcaoIndependeDosAcessos(t *testing.T) {
	destino := &destinoFalso{palavras: map[uint32]uint32{}, falha: true}
	s := NovoSistema(destino)
	s.AdicionarFonte(&fonteFalsa{pedindo: true})

	if _, err := s.Read32(0x100); err == nil {
		t.Fatal("a leitura deveria falhar")
	}
	if !s.Interrupcao() {
		t.Error("a interrupção é um fio separado: não depende dos acessos à memória")
	}
}
