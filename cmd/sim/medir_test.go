package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"riscv/desempenho"
	"riscv/programas"
)

// contem confere que o texto tem cada um dos trechos.
func contem(t *testing.T, texto string, trechos ...string) {
	t.Helper()
	for _, trecho := range trechos {
		if !strings.Contains(texto, trecho) {
			t.Errorf("deveria conter %q:\n%s", trecho, texto)
		}
	}
}

// naoContem confere que o texto não tem nenhum dos trechos.
func naoContem(t *testing.T, texto string, trechos ...string) {
	t.Helper()
	for _, trecho := range trechos {
		if strings.Contains(texto, trecho) {
			t.Errorf("não deveria conter %q:\n%s", trecho, texto)
		}
	}
}

// medicao monta um Resultado à mão, só com o que a comparação usa.
func medicao(minimo, maximo time.Duration, hits, misses int) desempenho.Resultado {
	r := desempenho.Resultado{
		Programa:   "teste",
		Instrucoes: 1000,
		Minimo:     minimo,
		Maximo:     maximo,
		Hits:       hits,
		Misses:     misses,
	}
	r.TaxaAcerto = float64(hits) / float64(hits+misses)
	return r
}

// escreverPrograma grava um programa num arquivo de texto e devolve o caminho.
func escreverPrograma(t *testing.T, dir string, palavras []uint32) string {
	t.Helper()
	caminho := filepath.Join(dir, "alfabeto.hex")
	arquivo, err := os.Create(caminho)
	if err != nil {
		t.Fatalf("erro ao criar o arquivo: %v", err)
	}
	defer arquivo.Close()
	if err := desempenho.EscreverPalavras(arquivo, palavras); err != nil {
		t.Fatalf("erro ao escrever o programa: %v", err)
	}
	return caminho
}

// TestFormatarDuracao confere as unidades e as casas decimais.
func TestFormatarDuracao(t *testing.T) {
	casos := []struct {
		duracao  time.Duration
		esperado string
	}{
		{2500 * time.Microsecond, "2.50 ms"},
		{time.Millisecond, "1.00 ms"},
		{2 * time.Second, "2000.00 ms"},
		{999 * time.Microsecond, "999.0 us"},
		{1500 * time.Nanosecond, "1.5 us"},
		{0, "0.0 us"},
	}

	for _, c := range casos {
		if obtido := formatarDuracao(c.duracao); obtido != c.esperado {
			t.Errorf("formatarDuracao(%v) = %q, esperado %q", c.duracao, obtido, c.esperado)
		}
	}
}

// TestFormatarResultado confere o resumo nos três casos de validação.
func TestFormatarResultado(t *testing.T) {
	ms := time.Millisecond
	r := desempenho.Resultado{
		Programa: "teste", Instrucoes: 1000,
		Minimo: 2 * ms, Mediana: 2500 * time.Microsecond, Maximo: 3 * ms,
		NsPorInstr: 2000, MIPS: 0.5,
		Hits: 800, Misses: 200, TaxaAcerto: 0.8,
	}

	// com resultado esperado e passando
	comEsperado := desempenho.Programa{Esperado: desempenho.Esperado{Instrucoes: 1000}}
	saida := formatarResultado(comEsperado, r)
	contem(t, saida, "teste", "1000", "2.00 ms / 2.50 ms / 3.00 ms", "2000.00 ns",
		"800 hits, 200 misses", "80.0% de acerto", "PASSOU")

	// com resultado esperado e falhando
	r.Falhas = []string{"x9 = 1, esperado 2"}
	saida = formatarResultado(comEsperado, r)
	contem(t, saida, "FALHOU", "- x9 = 1, esperado 2")
	naoContem(t, saida, "PASSOU")

	// sem resultado esperado: não dá para validar
	r.Falhas = nil
	saida = formatarResultado(desempenho.Programa{}, r)
	contem(t, saida, "não há resultado esperado")
	naoContem(t, saida, "PASSOU", "FALHOU")
}

// TestFormatarComparacao confere o texto de cada situação da comparação.
func TestFormatarComparacao(t *testing.T) {
	data := time.Date(2026, 11, 27, 10, 30, 0, 0, time.UTC)
	anterior := func(r desempenho.Resultado) desempenho.Registro {
		return desempenho.Registro{Rotulo: "linha de base", Data: data, Resultado: r}
	}
	ms := time.Millisecond

	// mais rápido, com menos misses
	antes := medicao(100*ms, 105*ms, 800, 200)
	depois := medicao(50*ms, 52*ms, 900, 100)
	saida := formatarComparacao(anterior(antes), desempenho.Comparar(antes, depois))
	contem(t, saida, `"linha de base"`, "27/11/2026 10:30", "2.00x mais rápido", "-50.0% no tempo",
		"misses: 200 -> 100 (-100, -50.0%)", "+10.00 pontos")
	naoContem(t, saida, "dentro do ruído", "ALERTA")

	// mais lento
	antes = medicao(50*ms, 52*ms, 900, 100)
	depois = medicao(100*ms, 102*ms, 700, 300)
	saida = formatarComparacao(anterior(antes), desempenho.Comparar(antes, depois))
	contem(t, saida, "2.00x mais lento", "+100.0% no tempo", "misses: 100 -> 300 (+200, +200.0%)")

	// diferença pequena, dentro do ruído
	antes = medicao(100*ms, 110*ms, 800, 200)
	depois = medicao(97*ms, 100*ms, 800, 200)
	saida = formatarComparacao(anterior(antes), desempenho.Comparar(antes, depois))
	contem(t, saida, "dentro do ruído", "-3.0%", "pode ser sorte")

	// o comportamento mudou
	antes = medicao(100*ms, 105*ms, 800, 200)
	depois = antes
	depois.Registradores[9] = 1
	saida = formatarComparacao(anterior(antes), desempenho.Comparar(antes, depois))
	contem(t, saida, "ALERTA", "quebrou o programa")

	// programas diferentes
	depois = medicao(100*ms, 105*ms, 800, 200)
	depois.Programa = "outro"
	saida = formatarComparacao(anterior(antes), desempenho.Comparar(antes, depois))
	contem(t, saida, "não são o mesmo programa")
}

// TestMedirDesempenhoPadrao confere o caminho completo com o programa padrão.
func TestMedirDesempenhoPadrao(t *testing.T) {
	historico := filepath.Join(t.TempDir(), "historico.json")
	var saida bytes.Buffer

	err := medirDesempenho(&saida, opcoesMedicao{rotulo: "linha de base", historico: historico, repeticoes: 2})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	contem(t, saida.String(), `Medindo "Soma de vetor" com 2 repetições`, "repetição 2 de 2",
		"153207 hits, 2081 misses", "PASSOU", "guardada em", historico)
	naoContem(t, saida.String(), "Comparação")

	registros, err := desempenho.Carregar(historico)
	if err != nil || len(registros) != 1 || registros[0].Rotulo != "linha de base" {
		t.Errorf("histórico = %v, %v; esperado 1 registro com o rótulo \"linha de base\"", registros, err)
	}
}

// TestMedirDesempenhoComparaComAAnterior confere que a segunda medição do
// mesmo programa é comparada com a primeira, e a terceira com a segunda.
func TestMedirDesempenhoComparaComAAnterior(t *testing.T) {
	dir := t.TempDir()
	programa := escreverPrograma(t, dir, programas.Alfabeto)
	historico := filepath.Join(dir, "historico.json")

	medir := func(rotulo string) string {
		t.Helper()
		var saida bytes.Buffer
		err := medirDesempenho(&saida, opcoesMedicao{
			rotulo: rotulo, programa: programa, historico: historico, repeticoes: 2,
		})
		if err != nil {
			t.Fatalf("%s: erro inesperado: %v", rotulo, err)
		}
		return saida.String()
	}

	primeira := medir("primeira")
	contem(t, primeira, `Medindo "alfabeto"`, "não há resultado esperado")
	naoContem(t, primeira, "Comparação")

	segunda := medir("segunda")
	contem(t, segunda, `Comparação com a medição anterior ("primeira"`)

	terceira := medir("terceira")
	contem(t, terceira, `Comparação com a medição anterior ("segunda"`)

	registros, err := desempenho.Carregar(historico)
	if err != nil || len(registros) != 3 {
		t.Fatalf("histórico = %d registros, %v; esperado 3", len(registros), err)
	}
	for i, rotulo := range []string{"primeira", "segunda", "terceira"} {
		if registros[i].Rotulo != rotulo {
			t.Errorf("registro %d = %q, esperado %q", i, registros[i].Rotulo, rotulo)
		}
	}
}

// TestMedirDesempenhoErros confere as falhas que não podem estragar nada.
func TestMedirDesempenhoErros(t *testing.T) {
	dir := t.TempDir()
	programa := escreverPrograma(t, dir, programas.Alfabeto)
	historico := filepath.Join(dir, "historico.json")

	// programa que não existe
	var saida bytes.Buffer
	err := medirDesempenho(&saida, opcoesMedicao{
		rotulo: "x", programa: filepath.Join(dir, "nao_existe.hex"), historico: historico,
	})
	if err == nil || !strings.Contains(err.Error(), "ao ler o programa") {
		t.Errorf("programa inexistente: erro = %v, esperado citar \"ao ler o programa\"", err)
	}

	// histórico inválido: o erro vem antes da medição e o arquivo fica como estava
	invalido := filepath.Join(dir, "invalido.json")
	if err := os.WriteFile(invalido, []byte("lixo"), 0o644); err != nil {
		t.Fatalf("erro ao criar o arquivo: %v", err)
	}
	saida.Reset()
	err = medirDesempenho(&saida, opcoesMedicao{rotulo: "x", programa: programa, historico: invalido})
	if err == nil || !strings.Contains(err.Error(), "ao ler o histórico") {
		t.Errorf("histórico inválido: erro = %v, esperado citar \"ao ler o histórico\"", err)
	}
	naoContem(t, saida.String(), "Medindo")
	if lido, _ := os.ReadFile(invalido); string(lido) != "lixo" {
		t.Errorf("o histórico inválido foi alterado: %q", lido)
	}

	// sem número de repetições, usa o padrão (5)
	saida.Reset()
	err = medirDesempenho(&saida, opcoesMedicao{rotulo: "x", programa: programa, historico: historico})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	contem(t, saida.String(), "com 5 repetições")
}
