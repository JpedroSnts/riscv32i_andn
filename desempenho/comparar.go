package desempenho

import "math"

// Comparacao é a diferença entre duas medições, pronta para a interface mostrar.
type Comparacao struct {
	MesmoPrograma  bool // mesmo nome e mesma quantidade de instruções
	MesmoResultado bool // mesmas instruções e mesmos registradores finais

	Aceleracao       float64 // tempo antes / tempo depois (2 = o dobro da velocidade)
	VariacaoTempoPct float64 // negativo = ficou mais rápido

	VariacaoMisses     int     // depois - antes (negativo = menos misses)
	VariacaoMissesPct  float64 // em porcentagem dos misses de antes
	VariacaoTaxaAcerto float64 // em pontos percentuais

	Ruido         float64 // maior oscilação entre o menor e o maior tempo das duas medições, em %
	DentroDoRuido bool    // a diferença de tempo é menor que o ruído: pode ser sorte
}

// Comparar compara duas medições, usando o tempo mínimo de cada uma.
// MesmoPrograma false significa que os tempos não são comparáveis.
// MesmoResultado false significa que o comportamento mudou: numa otimização, isso indica um erro.
func Comparar(antes, depois Resultado) Comparacao {
	c := Comparacao{
		MesmoPrograma:  antes.Programa == depois.Programa && antes.Instrucoes == depois.Instrucoes,
		MesmoResultado: antes.Instrucoes == depois.Instrucoes && antes.Registradores == depois.Registradores,
	}

	if antes.Minimo > 0 && depois.Minimo > 0 {
		c.Aceleracao = float64(antes.Minimo) / float64(depois.Minimo)
		c.VariacaoTempoPct = (float64(depois.Minimo) - float64(antes.Minimo)) / float64(antes.Minimo) * 100
	}

	c.VariacaoMisses = depois.Misses - antes.Misses
	if antes.Misses > 0 {
		c.VariacaoMissesPct = float64(c.VariacaoMisses) / float64(antes.Misses) * 100
	}
	c.VariacaoTaxaAcerto = (depois.TaxaAcerto - antes.TaxaAcerto) * 100

	c.Ruido = math.Max(ruido(antes), ruido(depois))
	c.DentroDoRuido = math.Abs(c.VariacaoTempoPct) <= c.Ruido
	return c
}

// ruido é quanto o maior tempo passa do menor, em porcentagem.
func ruido(r Resultado) float64 {
	if r.Minimo <= 0 {
		return 0
	}
	return float64(r.Maximo-r.Minimo) / float64(r.Minimo) * 100
}
