package desempenho

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestHistoricoSalvarECarregar confere que o que se salva volta igual e em ordem.
func TestHistoricoSalvarECarregar(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "historico.json")
	data := time.Date(2026, 11, 27, 10, 30, 0, 0, time.UTC)
	ms := time.Millisecond

	primeiro := Registro{
		Rotulo: "antes",
		Data:   data,
		Resultado: Resultado{
			Programa:   "teste",
			Instrucoes: 1000,
			Duracoes:   []time.Duration{100 * ms, 105 * ms, 110 * ms},
			Minimo:     100 * ms,
			Maximo:     110 * ms,
			Hits:       800,
			Misses:     200,
		},
	}
	primeiro.Resultado.Registradores[9] = 3289600
	segundo := Registro{
		Rotulo:    "depois",
		Data:      data.Add(time.Hour),
		Resultado: Resultado{Programa: "teste", Instrucoes: 1000, Minimo: 50 * ms},
	}

	for _, registro := range []Registro{primeiro, segundo} {
		if err := Salvar(caminho, registro); err != nil {
			t.Fatalf("erro ao salvar %q: %v", registro.Rotulo, err)
		}
	}

	registros, err := Carregar(caminho)
	if err != nil {
		t.Fatalf("erro ao carregar: %v", err)
	}
	if len(registros) != 2 {
		t.Fatalf("carregou %d registros, esperado 2", len(registros))
	}

	if registros[0].Rotulo != "antes" || registros[1].Rotulo != "depois" {
		t.Errorf("rótulos = %q e %q, esperado antes e depois", registros[0].Rotulo, registros[1].Rotulo)
	}
	if !registros[0].Data.Equal(data) {
		t.Errorf("Data = %v, esperado %v", registros[0].Data, data)
	}
	r := registros[0].Resultado
	if r.Minimo != 100*ms || r.Hits != 800 || r.Misses != 200 {
		t.Errorf("Minimo = %v, Hits = %d, Misses = %d", r.Minimo, r.Hits, r.Misses)
	}
	if len(r.Duracoes) != 3 || r.Duracoes[1] != 105*ms {
		t.Errorf("Duracoes = %v, esperado as três medições", r.Duracoes)
	}
	if r.Registradores[9] != 3289600 {
		t.Errorf("x9 = %d, esperado 3289600", r.Registradores[9])
	}
}

// TestHistoricoInexistenteOuVazio confere que ainda não ter histórico não é erro.
func TestHistoricoInexistenteOuVazio(t *testing.T) {
	dir := t.TempDir()

	registros, err := Carregar(filepath.Join(dir, "nao_existe.json"))
	if err != nil || len(registros) != 0 {
		t.Errorf("arquivo inexistente: %v, %v; esperado lista vazia e sem erro", registros, err)
	}

	vazio := filepath.Join(dir, "vazio.json")
	if err := os.WriteFile(vazio, []byte("  \n"), 0o644); err != nil {
		t.Fatalf("erro ao criar o arquivo: %v", err)
	}
	registros, err = Carregar(vazio)
	if err != nil || len(registros) != 0 {
		t.Errorf("arquivo vazio: %v, %v; esperado lista vazia e sem erro", registros, err)
	}
}

// TestHistoricoInvalidoNaoEhSobrescrito confere que um arquivo ilegível é preservado.
func TestHistoricoInvalidoNaoEhSobrescrito(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "historico.json")
	conteudo := "isto não é json"
	if err := os.WriteFile(caminho, []byte(conteudo), 0o644); err != nil {
		t.Fatalf("erro ao criar o arquivo: %v", err)
	}

	if _, err := Carregar(caminho); err == nil {
		t.Error("Carregar deveria dar erro num arquivo inválido")
	}
	if err := Salvar(caminho, Registro{Rotulo: "x"}); err == nil {
		t.Error("Salvar deveria dar erro num arquivo inválido")
	}

	lido, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("erro ao ler: %v", err)
	}
	if string(lido) != conteudo {
		t.Errorf("o arquivo foi alterado: %q", lido)
	}
}

// TestHistoricoSemDataUsaAgora confere a data automática e que não sobra arquivo temporário.
func TestHistoricoSemDataUsaAgora(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "historico.json")

	if err := Salvar(caminho, Registro{Rotulo: "agora"}); err != nil {
		t.Fatalf("erro ao salvar: %v", err)
	}
	registros, err := Carregar(caminho)
	if err != nil || len(registros) != 1 {
		t.Fatalf("Carregar = %v, %v; esperado 1 registro", registros, err)
	}

	if registros[0].Data.IsZero() || time.Since(registros[0].Data) > time.Minute {
		t.Errorf("Data = %v, esperado a hora atual", registros[0].Data)
	}
	if _, err := os.Stat(caminho + ".tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Error("não deveria sobrar o arquivo temporário")
	}
}
