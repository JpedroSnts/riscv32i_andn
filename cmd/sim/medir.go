package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"riscv/desempenho"
)

// opcoesMedicao reúne o que o modo -medir recebe da linha de comando.
type opcoesMedicao struct {
	rotulo     string // nome da medição no histórico (por exemplo "linha de base")
	programa   string // arquivo com o programa; vazio = soma de vetor
	historico  string // arquivo do histórico de medições
	repeticoes int    // quantas vezes o programa é executado e medido
}

// medirDesempenho mede um programa, escreve o resultado em saida, guarda a medição no histórico e compara com a anterior do mesmo programa.
func medirDesempenho(saida io.Writer, o opcoesMedicao) error {
	if o.repeticoes <= 0 {
		o.repeticoes = 5
	}

	programa := desempenho.Padrao()[0] // soma de vetor
	if o.programa != "" {
		lido, err := desempenho.LerArquivo(o.programa)
		if err != nil {
			return fmt.Errorf("ao ler o programa: %w", err)
		}
		programa = lido
	}

	// lê o histórico antes de medir: se o arquivo estiver inválido, o erro aparece logo
	registros, err := desempenho.Carregar(o.historico)
	if err != nil {
		return fmt.Errorf("ao ler o histórico: %w", err)
	}

	fmt.Fprintf(saida, "Medindo %q com %d repetições...\n", programa.Nome, o.repeticoes)
	r, err := desempenho.Medir(context.Background(), programa, desempenho.Opcoes{
		Repeticoes: o.repeticoes,
		Progresso: func(feitas, total int) {
			fmt.Fprintf(saida, "  repetição %d de %d\n", feitas, total)
		},
	})
	if err != nil {
		return fmt.Errorf("na medição: %w", err)
	}

	fmt.Fprint(saida, formatarResultado(programa, r))

	if !r.Passou() {
		return errors.New("o programa não terminou como esperado: a medição não foi guardada")
	}

	// a anterior é buscada antes de guardar a nova, senão ela se compararia consigo mesma
	anterior, temAnterior := desempenho.UltimoDoMesmoPrograma(registros, r)

	if err := desempenho.Salvar(o.historico, desempenho.Registro{Rotulo: o.rotulo, Resultado: r}); err != nil {
		return fmt.Errorf("ao guardar a medição: %w", err)
	}
	caminho, err := filepath.Abs(o.historico)
	if err != nil {
		caminho = o.historico
	}
	fmt.Fprintf(saida, "\nMedição %q guardada em %s\n", o.rotulo, caminho)

	if temAnterior {
		fmt.Fprint(saida, formatarComparacao(anterior, desempenho.Comparar(anterior.Resultado, r)))
	}
	return nil
}

// formatarResultado monta o resumo de uma medição.
func formatarResultado(programa desempenho.Programa, r desempenho.Resultado) string {
	var s strings.Builder

	fmt.Fprintf(&s, "\nResultado: %s\n", r.Programa)
	fmt.Fprintf(&s, "  instruções executadas: %d\n", r.Instrucoes)
	fmt.Fprintf(&s, "  tempo (mínimo / mediana / máximo): %s / %s / %s\n",
		formatarDuracao(r.Minimo), formatarDuracao(r.Mediana), formatarDuracao(r.Maximo))
	fmt.Fprintf(&s, "  por instrução: %.2f ns (%.1f milhões de instruções por segundo)\n", r.NsPorInstr, r.MIPS)
	fmt.Fprintf(&s, "  cache: %d hits, %d misses (%.1f%% de acerto)\n", r.Hits, r.Misses, r.TaxaAcerto*100)

	switch {
	case !temEsperado(programa.Esperado):
		s.WriteString("  validação: não há resultado esperado para este programa\n")
	case r.Passou():
		s.WriteString("  validação: PASSOU\n")
	default:
		s.WriteString("  validação: FALHOU\n")
		for _, falha := range r.Falhas {
			fmt.Fprintf(&s, "    - %s\n", falha)
		}
	}
	return s.String()
}

// temEsperado diz se o programa traz um resultado para conferir.
func temEsperado(e desempenho.Esperado) bool {
	return e.Instrucoes != 0 || len(e.Registradores) > 0
}

// formatarComparacao monta o texto da comparação com a medição anterior.
func formatarComparacao(anterior desempenho.Registro, c desempenho.Comparacao) string {
	var s strings.Builder

	fmt.Fprintf(&s, "\nComparação com a medição anterior (%q, %s):\n",
		anterior.Rotulo, anterior.Data.Format("02/01/2006 15:04"))

	if !c.MesmoPrograma {
		s.WriteString("  aviso: não são o mesmo programa, então os tempos não são comparáveis\n")
	}

	switch {
	case c.Aceleracao == 0:
		s.WriteString("  tempo: sem tempos para comparar\n")
	case c.VariacaoTempoPct < 0:
		fmt.Fprintf(&s, "  tempo: %.2fx mais rápido (%+.1f%% no tempo)\n", c.Aceleracao, c.VariacaoTempoPct)
	case c.VariacaoTempoPct > 0:
		fmt.Fprintf(&s, "  tempo: %.2fx mais lento (%+.1f%% no tempo)\n", 1/c.Aceleracao, c.VariacaoTempoPct)
	default:
		s.WriteString("  tempo: igual\n")
	}

	antes := anterior.Resultado.Misses
	fmt.Fprintf(&s, "  misses: %d -> %d (%+d, %+.1f%%)\n", antes, antes+c.VariacaoMisses, c.VariacaoMisses, c.VariacaoMissesPct)
	fmt.Fprintf(&s, "  taxa de acerto: %+.2f pontos percentuais\n", c.VariacaoTaxaAcerto)

	if c.Aceleracao != 0 && c.DentroDoRuido {
		fmt.Fprintf(&s, "  aviso: a diferença de tempo (%+.1f%%) está dentro do ruído das medições (%.1f%%), então pode ser sorte\n",
			c.VariacaoTempoPct, c.Ruido)
	}
	if !c.MesmoResultado {
		s.WriteString("  ALERTA: o resultado final do programa mudou (instruções ou registradores diferentes).\n")
		s.WriteString("          Se isto é uma otimização, ela quebrou o programa.\n")
	}
	return s.String()
}

// formatarDuracao mostra o tempo em ms ou us, com casas decimais.
func formatarDuracao(d time.Duration) string {
	if d >= time.Millisecond {
		return fmt.Sprintf("%.2f ms", float64(d)/float64(time.Millisecond))
	}
	return fmt.Sprintf("%.1f us", float64(d)/float64(time.Microsecond))
}
