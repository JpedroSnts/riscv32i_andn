package desempenho

import (
	"context"
	"math"
	"testing"
	"time"
)

const milissegundo = time.Millisecond

// proximo compara números decimais com uma tolerância pequena.
func proximo(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// resultadoDe monta um Resultado à mão, só com o que a comparação usa.
func resultadoDe(minimo, mediana time.Duration, hits, misses int) Resultado {
	r := Resultado{
		Programa:   "teste",
		Instrucoes: 1000,
		Minimo:     minimo,
		Mediana:    mediana,
		Maximo:     mediana,
		Hits:       hits,
		Misses:     misses,
	}
	if total := hits + misses; total > 0 {
		r.TaxaAcerto = float64(hits) / float64(total)
	}
	return r
}

// TestCompararMaisRapido confere uma otimização que dobra a velocidade e reduz misses.
func TestCompararMaisRapido(t *testing.T) {
	antes := resultadoDe(100*milissegundo, 105*milissegundo, 800, 200)
	depois := resultadoDe(50*milissegundo, 52*milissegundo, 900, 100)

	c := Comparar(antes, depois)

	if !proximo(c.Aceleracao, 2) {
		t.Errorf("Aceleracao = %v, esperado 2", c.Aceleracao)
	}
	if !proximo(c.VariacaoTempoPct, -50) {
		t.Errorf("VariacaoTempoPct = %v, esperado -50", c.VariacaoTempoPct)
	}
	if c.VariacaoMisses != -100 || !proximo(c.VariacaoMissesPct, -50) {
		t.Errorf("VariacaoMisses = %d (%v%%), esperado -100 (-50%%)", c.VariacaoMisses, c.VariacaoMissesPct)
	}
	if !proximo(c.VariacaoTaxaAcerto, 10) {
		t.Errorf("VariacaoTaxaAcerto = %v, esperado 10 pontos", c.VariacaoTaxaAcerto)
	}
	if !proximo(c.Ruido, 5) { // o maior entre 5% (antes) e 4% (depois)
		t.Errorf("Ruido = %v, esperado 5", c.Ruido)
	}
	if c.DentroDoRuido {
		t.Error("um ganho de 50% com ruído de 5% não é ruído")
	}
	if !c.MesmoPrograma || !c.MesmoResultado {
		t.Errorf("MesmoPrograma = %v, MesmoResultado = %v, esperado true e true", c.MesmoPrograma, c.MesmoResultado)
	}
}

// TestCompararDentroDoRuido confere que uma diferença pequena é marcada como possível sorte.
func TestCompararDentroDoRuido(t *testing.T) {
	antes := resultadoDe(100*milissegundo, 110*milissegundo, 800, 200) // ruído de 10%
	depois := resultadoDe(97*milissegundo, 100*milissegundo, 800, 200)

	c := Comparar(antes, depois)

	if !proximo(c.VariacaoTempoPct, -3) {
		t.Errorf("VariacaoTempoPct = %v, esperado -3", c.VariacaoTempoPct)
	}
	if !c.DentroDoRuido {
		t.Errorf("-3%% com ruído de %v%% deveria estar dentro do ruído", c.Ruido)
	}
}

// TestCompararMaisLento confere uma mudança que piora o tempo e os misses.
func TestCompararMaisLento(t *testing.T) {
	antes := resultadoDe(50*milissegundo, 52*milissegundo, 900, 100)
	depois := resultadoDe(100*milissegundo, 102*milissegundo, 700, 300)

	c := Comparar(antes, depois)

	if !proximo(c.Aceleracao, 0.5) {
		t.Errorf("Aceleracao = %v, esperado 0.5", c.Aceleracao)
	}
	if !proximo(c.VariacaoTempoPct, 100) {
		t.Errorf("VariacaoTempoPct = %v, esperado 100", c.VariacaoTempoPct)
	}
	if c.VariacaoMisses != 200 || !proximo(c.VariacaoMissesPct, 200) {
		t.Errorf("VariacaoMisses = %d (%v%%), esperado 200 (200%%)", c.VariacaoMisses, c.VariacaoMissesPct)
	}
	if c.DentroDoRuido {
		t.Error("dobrar o tempo não é ruído")
	}
}

// TestCompararNaoComparaveis confere os avisos de programa e comportamento diferentes.
func TestCompararNaoComparaveis(t *testing.T) {
	base := resultadoDe(100*milissegundo, 105*milissegundo, 800, 200)

	// outro programa
	outro := base
	outro.Programa = "outro"
	if c := Comparar(base, outro); c.MesmoPrograma {
		t.Error("nomes diferentes: MesmoPrograma deveria ser false")
	}

	// outra quantidade de instruções
	menos := base
	menos.Instrucoes = 999
	if c := Comparar(base, menos); c.MesmoPrograma || c.MesmoResultado {
		t.Errorf("instruções diferentes: MesmoPrograma = %v, MesmoResultado = %v, esperado false e false",
			c.MesmoPrograma, c.MesmoResultado)
	}

	// mesmo programa e instruções, mas um registrador final diferente: o comportamento mudou
	mudou := base
	mudou.Registradores[9] = 1
	c := Comparar(base, mudou)
	if !c.MesmoPrograma {
		t.Error("mesmo programa e instruções: MesmoPrograma deveria ser true")
	}
	if c.MesmoResultado {
		t.Error("um registrador final diferente: MesmoResultado deveria ser false")
	}
}

// TestCompararSemTempo confere que medições vazias não geram divisão por zero.
func TestCompararSemTempo(t *testing.T) {
	c := Comparar(Resultado{}, Resultado{})

	for nome, valor := range map[string]float64{
		"Aceleracao":         c.Aceleracao,
		"VariacaoTempoPct":   c.VariacaoTempoPct,
		"VariacaoMissesPct":  c.VariacaoMissesPct,
		"VariacaoTaxaAcerto": c.VariacaoTaxaAcerto,
		"Ruido":              c.Ruido,
	} {
		if math.IsNaN(valor) || math.IsInf(valor, 0) || valor != 0 {
			t.Errorf("%s = %v, esperado 0", nome, valor)
		}
	}
}

// TestCompararComMedicaoReal compara uma medição de verdade com ela mesma: nada pode ter mudado.
func TestCompararComMedicaoReal(t *testing.T) {
	r, err := Medir(context.Background(), Padrao()[0], Opcoes{Repeticoes: 2})
	if err != nil {
		t.Fatalf("erro na medição: %v", err)
	}

	c := Comparar(r, r)

	if !c.MesmoPrograma || !c.MesmoResultado {
		t.Errorf("MesmoPrograma = %v, MesmoResultado = %v, esperado true e true", c.MesmoPrograma, c.MesmoResultado)
	}
	if c.Aceleracao != 1 || c.VariacaoTempoPct != 0 || c.VariacaoMisses != 0 {
		t.Errorf("Aceleracao = %v, VariacaoTempoPct = %v, VariacaoMisses = %d; esperado 1, 0 e 0",
			c.Aceleracao, c.VariacaoTempoPct, c.VariacaoMisses)
	}
	if !c.DentroDoRuido {
		t.Error("uma medição comparada com ela mesma não tem diferença: deveria estar dentro do ruído")
	}
}

// TestRuidoIgnoraUmaExecucaoAtrapalhada confere que um único tempo fora da curva não infla o ruído: um ganho real de 15% tem que aparecer como ganho.
func TestRuidoIgnoraUmaExecucaoAtrapalhada(t *testing.T) {
	ms := time.Millisecond
	antes := resultadoDe(100*ms, 102*ms, 800, 200)
	antes.Maximo = 300 * ms // uma única execução atrapalhada pelo sistema
	depois := resultadoDe(85*ms, 87*ms, 800, 200)

	c := Comparar(antes, depois)

	if !proximo(c.VariacaoTempoPct, -15) {
		t.Errorf("VariacaoTempoPct = %v, esperado -15", c.VariacaoTempoPct)
	}
	if c.Ruido > 3 {
		t.Errorf("Ruido = %v, o tempo máximo fora da curva não deveria contar", c.Ruido)
	}
	if c.DentroDoRuido {
		t.Error("um ganho de 15% com ruído de uns 2% não é sorte")
	}
}
