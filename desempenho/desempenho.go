package desempenho

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"time"

	"riscv/programas"
	"riscv/simulador"
)

// Esperado descreve o estado final correto de um programa. Serve para provar
// que uma otimização não mudou o comportamento.
type Esperado struct {
	Instrucoes    int            // quantas instruções o programa executa (0 = não confere)
	Registradores map[int]uint32 // valor final de cada registrador conferido
}

// Programa é o que o benchmark executa.
type Programa struct {
	Nome     string
	Palavras []uint32 // instruções, carregadas a partir do endereço 0
	Esperado Esperado
}

// Opcoes controla a medição. Os campos zerados viram valores padrão.
type Opcoes struct {
	Repeticoes   int                     // execuções medidas (padrão 5)
	LimitePassos int                     // trava contra laço infinito (padrão 100 milhões)
	Progresso    func(feitas, total int) // chamada entre as repetições, nunca durante uma
}

// Resultado reúne tudo o que a interface precisa mostrar.
type Resultado struct {
	Programa   string
	Repeticoes int
	Duracoes   []time.Duration // de cada repetição, na ordem em que rodaram

	Minimo, Mediana, Maximo time.Duration

	Instrucoes int
	NsPorInstr float64 // nanossegundos por instrução, com o tempo mínimo
	MIPS       float64 // milhões de instruções por segundo, com o tempo mínimo

	Hits, Misses int
	TaxaAcerto   float64 // hits / (hits + misses), de 0 a 1

	Registradores [32]uint32 // estado final
	Falhas        []string   // vazio = o programa terminou como esperado
}

// Passou diz se o programa terminou como esperado.
func (r Resultado) Passou() bool {
	return len(r.Falhas) == 0
}

// Padrao devolve os programas de benchmark que já vêm prontos.
func Padrao() []Programa {
	return []Programa{
		{
			Nome:     "Soma de vetor",
			Palavras: programas.SomaVetor,
			Esperado: Esperado{Instrucoes: 129432, Registradores: map[int]uint32{9: 3289600}},
		},
		{
			Nome:     "Alfabeto",
			Palavras: programas.Alfabeto,
			Esperado: Esperado{Instrucoes: 57, Registradores: map[int]uint32{8: 0}},
		},
	}
}

// Medir executa o programa várias vezes e devolve os tempos e as estatísticas. Pode ser chamada numa goroutine; cancelar o ctx interrompe entre repetições.
func Medir(ctx context.Context, programa Programa, opcoes Opcoes) (Resultado, error) {
	if opcoes.Repeticoes <= 0 {
		opcoes.Repeticoes = 5
	}
	if opcoes.LimitePassos <= 0 {
		opcoes.LimitePassos = 100_000_000
	}
	if err := ctx.Err(); err != nil {
		return Resultado{}, err
	}

	// aquecimento: uma execução que não conta
	if _, _, _, err := executarUma(programa, opcoes.LimitePassos); err != nil {
		return Resultado{}, fmt.Errorf("aquecimento: %w", err)
	}

	r := Resultado{Programa: programa.Nome, Repeticoes: opcoes.Repeticoes}
	var ultima *simulador.Simulador

	for i := 0; i < opcoes.Repeticoes; i++ {
		if err := ctx.Err(); err != nil {
			return Resultado{}, err
		}

		duracao, sim, passos, err := executarUma(programa, opcoes.LimitePassos)
		if err != nil {
			return Resultado{}, err
		}
		if i > 0 && passos != r.Instrucoes {
			return Resultado{}, fmt.Errorf("execução não determinística: %d e %d instruções", r.Instrucoes, passos)
		}

		r.Duracoes = append(r.Duracoes, duracao)
		r.Instrucoes = passos
		ultima = sim

		if opcoes.Progresso != nil {
			opcoes.Progresso(i+1, opcoes.Repeticoes)
		}
	}

	// estatísticas dos tempos
	ordenadas := append([]time.Duration(nil), r.Duracoes...)
	sort.Slice(ordenadas, func(a, b int) bool { return ordenadas[a] < ordenadas[b] })
	r.Minimo = ordenadas[0]
	r.Maximo = ordenadas[len(ordenadas)-1]
	r.Mediana = mediana(ordenadas)

	if r.Minimo > 0 && r.Instrucoes > 0 {
		r.NsPorInstr = float64(r.Minimo.Nanoseconds()) / float64(r.Instrucoes)
		r.MIPS = float64(r.Instrucoes) / r.Minimo.Seconds() / 1e6
	}

	// o estado final e a cache são iguais em todas as repetições
	r.Hits, r.Misses = ultima.Cache.Hits, ultima.Cache.Misses
	if total := r.Hits + r.Misses; total > 0 {
		r.TaxaAcerto = float64(r.Hits) / float64(total)
	}
	r.Registradores = ultima.CPU.Regs
	r.Falhas = conferir(programa.Esperado, r)

	return r, nil
}

// executarUma monta uma máquina nova e mede só a execução do programa.
func executarUma(programa Programa, limite int) (time.Duration, *simulador.Simulador, int, error) {
	sim, err := simulador.Novo(programa.Palavras)
	if err != nil {
		return 0, nil, 0, err
	}

	runtime.GC() // a coleta de lixo acontece antes do relógio ligar, para não atrapalhar

	inicio := time.Now()
	passos, err := sim.Executar(limite, 0, nil) // sem callback: nada além da CPU entra na medição
	duracao := time.Since(inicio)
	if err != nil {
		return 0, nil, passos, err
	}
	return duracao, sim, passos, nil
}

// mediana recebe os tempos já ordenados.
func mediana(ordenadas []time.Duration) time.Duration {
	n := len(ordenadas)
	if n%2 == 1 {
		return ordenadas[n/2]
	}
	return (ordenadas[n/2-1] + ordenadas[n/2]) / 2
}

// conferir compara o resultado com o esperado e devolve as diferenças.
func conferir(esperado Esperado, r Resultado) []string {
	var falhas []string

	if esperado.Instrucoes != 0 && r.Instrucoes != esperado.Instrucoes {
		falhas = append(falhas, fmt.Sprintf("instruções executadas = %d, esperado %d",
			r.Instrucoes, esperado.Instrucoes))
	}

	// em ordem, para a mensagem sair sempre igual
	numeros := make([]int, 0, len(esperado.Registradores))
	for numero := range esperado.Registradores {
		numeros = append(numeros, numero)
	}
	sort.Ints(numeros)

	for _, numero := range numeros {
		if numero < 0 || numero >= 32 {
			falhas = append(falhas, fmt.Sprintf("o registrador x%d não existe", numero))
			continue
		}
		if obtido, quer := r.Registradores[numero], esperado.Registradores[numero]; obtido != quer {
			falhas = append(falhas, fmt.Sprintf("x%d = %d, esperado %d", numero, obtido, quer))
		}
	}
	return falhas
}
