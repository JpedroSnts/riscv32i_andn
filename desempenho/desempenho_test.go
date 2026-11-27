package desempenho

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"riscv/simulador"
)

// TestMedirSomaVetor confere o resultado de uma medição completa.
func TestMedirSomaVetor(t *testing.T) {
	r, err := Medir(context.Background(), Padrao()[0], Opcoes{Repeticoes: 3})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if !r.Passou() {
		t.Errorf("o programa deveria passar na validação: %v", r.Falhas)
	}
	if r.Instrucoes != 129432 {
		t.Errorf("Instrucoes = %d, esperado 129432", r.Instrucoes)
	}
	if r.Registradores[9] != 3289600 {
		t.Errorf("x9 = %d, esperado 3289600", r.Registradores[9])
	}
	if r.Hits != 153207 || r.Misses != 2081 {
		t.Errorf("hits/misses = %d/%d, esperado 153207/2081", r.Hits, r.Misses)
	}
	if r.Repeticoes != 3 || len(r.Duracoes) != 3 {
		t.Errorf("Repeticoes = %d, Duracoes = %d, esperado 3 e 3", r.Repeticoes, len(r.Duracoes))
	}
	if r.Minimo <= 0 || r.Minimo > r.Mediana || r.Mediana > r.Maximo {
		t.Errorf("tempos fora de ordem: mínimo %v, mediana %v, máximo %v", r.Minimo, r.Mediana, r.Maximo)
	}
	if r.NsPorInstr <= 0 || r.MIPS <= 0 {
		t.Errorf("NsPorInstr = %v, MIPS = %v, esperado valores positivos", r.NsPorInstr, r.MIPS)
	}
}

// TestTaxaDeAcerto confere a conta de hits sobre o total de acessos.
func TestTaxaDeAcerto(t *testing.T) {
	r, err := Medir(context.Background(), Padrao()[0], Opcoes{Repeticoes: 1})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	esperado := float64(153207) / float64(153207+2081)
	if math.Abs(r.TaxaAcerto-esperado) > 1e-9 {
		t.Errorf("TaxaAcerto = %v, esperado %v", r.TaxaAcerto, esperado)
	}
}

// TestMedirTodosOsPadroes confere que todo programa pronto passa na validação.
func TestMedirTodosOsPadroes(t *testing.T) {
	for _, programa := range Padrao() {
		r, err := Medir(context.Background(), programa, Opcoes{Repeticoes: 2})
		if err != nil {
			t.Errorf("%s: erro inesperado: %v", programa.Nome, err)
			continue
		}
		if !r.Passou() {
			t.Errorf("%s: falhas: %v", programa.Nome, r.Falhas)
		}
		if r.Programa != programa.Nome {
			t.Errorf("Programa = %q, esperado %q", r.Programa, programa.Nome)
		}
	}
}

// TestPadraoTemProgramas confere a lista de programas prontos.
func TestPadraoTemProgramas(t *testing.T) {
	lista := Padrao()
	if len(lista) < 2 {
		t.Fatalf("a lista tem %d programas, esperado pelo menos 2", len(lista))
	}

	nomes := map[string]bool{}
	for _, p := range lista {
		if p.Nome == "" || len(p.Palavras) == 0 {
			t.Errorf("programa sem nome ou sem instruções: %+v", p.Nome)
		}
		if nomes[p.Nome] {
			t.Errorf("nome repetido: %q", p.Nome)
		}
		nomes[p.Nome] = true
	}
}

// TestOpcoesPadrao confere que o que fica zerado vira o valor padrão.
func TestOpcoesPadrao(t *testing.T) {
	r, err := Medir(context.Background(), Padrao()[1], Opcoes{})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if r.Repeticoes != 5 || len(r.Duracoes) != 5 {
		t.Errorf("Repeticoes = %d, Duracoes = %d, esperado 5 e 5", r.Repeticoes, len(r.Duracoes))
	}
}

// TestValidacaoDetectaErros confere que um resultado diferente do esperado vira uma lista de falhas.
func TestValidacaoDetectaErros(t *testing.T) {
	programa := Padrao()[1] // alfabeto: 57 instruções, x8 = 0, x7 = 'i'
	programa.Esperado = Esperado{
		Instrucoes:    56,
		Registradores: map[int]uint32{8: 1, 7: 2},
	}

	r, err := Medir(context.Background(), programa, Opcoes{Repeticoes: 1})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if r.Passou() {
		t.Error("deveria falhar na validação")
	}
	if len(r.Falhas) != 3 {
		t.Errorf("Falhas = %v, esperado 3 mensagens", r.Falhas)
	}
}

// TestRegistradorInexistente confere a validação de um número de registrador inválido.
func TestRegistradorInexistente(t *testing.T) {
	programa := Padrao()[1]
	programa.Esperado = Esperado{Registradores: map[int]uint32{40: 0, -1: 0}}

	r, err := Medir(context.Background(), programa, Opcoes{Repeticoes: 1})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(r.Falhas) != 2 {
		t.Errorf("Falhas = %v, esperado 2 mensagens", r.Falhas)
	}
}

// TestLimiteDePassos confere que um laço infinito vira erro.
func TestLimiteDePassos(t *testing.T) {
	programa := Programa{Nome: "laço infinito", Palavras: []uint32{0x00000063}} // beq x0, x0, 0

	if _, err := Medir(context.Background(), programa, Opcoes{LimitePassos: 100}); err == nil {
		t.Error("deveria dar erro de limite de passos")
	}
}

// TestProgramaInvalido confere que uma instrução inválida vira erro.
func TestProgramaInvalido(t *testing.T) {
	programa := Programa{Nome: "inválido", Palavras: []uint32{0x00000000}}

	if _, err := Medir(context.Background(), programa, Opcoes{}); err == nil {
		t.Error("deveria dar erro de instrução inválida")
	}
}

// TestCancelamento confere que um contexto cancelado interrompe a medição.
func TestCancelamento(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()

	_, err := Medir(ctx, Padrao()[0], Opcoes{Repeticoes: 3})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("erro = %v, esperado context.Canceled", err)
	}
}

// TestProgresso confere que a função de progresso é chamada uma vez por repetição.
func TestProgresso(t *testing.T) {
	var chamadas [][2]int
	_, err := Medir(context.Background(), Padrao()[1], Opcoes{
		Repeticoes: 4,
		Progresso: func(feitas, total int) {
			chamadas = append(chamadas, [2]int{feitas, total})
		},
	})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	esperado := [][2]int{{1, 4}, {2, 4}, {3, 4}, {4, 4}}
	if len(chamadas) != len(esperado) {
		t.Fatalf("chamadas = %v, esperado %v", chamadas, esperado)
	}
	for i := range esperado {
		if chamadas[i] != esperado[i] {
			t.Errorf("chamada %d = %v, esperado %v", i, chamadas[i], esperado[i])
		}
	}
}

// TestMediana confere a mediana com quantidade ímpar e par de medições.
func TestMediana(t *testing.T) {
	ms := time.Millisecond
	casos := []struct {
		nome      string
		ordenadas []time.Duration
		esperado  time.Duration
	}{
		{"uma medição", []time.Duration{7 * ms}, 7 * ms},
		{"ímpar", []time.Duration{1 * ms, 2 * ms, 9 * ms}, 2 * ms},
		{"par", []time.Duration{10 * ms, 20 * ms}, 15 * ms},
		{"par com quatro", []time.Duration{1 * ms, 2 * ms, 4 * ms, 100 * ms}, 3 * ms},
	}

	for _, c := range casos {
		if obtido := mediana(c.ordenadas); obtido != c.esperado {
			t.Errorf("%s: mediana = %v, esperado %v", c.nome, obtido, c.esperado)
		}
	}
}

// BenchmarkSomaVetor mede quanto tempo o simulador leva para executar o programa de soma de vetor. É o mesmo programa da aba de benchmark.
func BenchmarkSomaVetor(b *testing.B) {
	programa := Padrao()[0]
	var instrucoes, hits, misses int

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer() // a montagem da máquina não entra na medição
		sim, err := simulador.Novo(programa.Palavras)
		if err != nil {
			b.Fatalf("erro ao montar: %v", err)
		}
		b.StartTimer()

		passos, err := sim.Executar(10_000_000, 0, nil)
		if err != nil {
			b.Fatalf("erro na execução: %v", err)
		}
		instrucoes, hits, misses = passos, sim.Cache.Hits, sim.Cache.Misses
	}

	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*instrucoes), "ns/instr")
	b.ReportMetric(float64(hits), "hits/op")
	b.ReportMetric(float64(misses), "misses/op")
}
