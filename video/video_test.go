package video

import (
	"errors"
	"strings"
	"testing"
)

// memoriaFalsa é um Barramento mínimo, só para os testes.
type memoriaFalsa struct {
	palavras map[uint32]uint32
	falhaEm  map[uint32]bool // endereços em que a leitura dá erro
}

var errFalso = errors.New("falha simulada")

func novaMemoria() *memoriaFalsa {
	return &memoriaFalsa{palavras: map[uint32]uint32{}, falhaEm: map[uint32]bool{}}
}

func (m *memoriaFalsa) Read32(endereco uint32) (uint32, error) {
	if m.falhaEm[endereco] {
		return 0, errFalso
	}
	return m.palavras[endereco], nil
}

func (m *memoriaFalsa) Write32(endereco uint32, valor uint32) error {
	m.palavras[endereco] = valor
	return nil
}

// enderecoDe devolve o endereço da célula (linha, coluna) da tela.
func enderecoDe(linha, coluna int) uint32 {
	return uint32(Base + (linha*Colunas+coluna)*4)
}

// linhasDe separa o texto da tela em linhas (a primeira e a última são a borda).
func linhasDe(tela string) []string {
	return strings.Split(strings.TrimSuffix(tela, "\n"), "\n")
}

// TestMontarTelaVazia confere o quadro de uma VRAM sem nada escrito.
func TestMontarTelaVazia(t *testing.T) {
	tela, err := Montar(novaMemoria())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	borda := "+" + strings.Repeat("-", Colunas) + "+\n"
	vazia := "|" + strings.Repeat(" ", Colunas) + "|\n"
	esperado := borda + strings.Repeat(vazia, Linhas) + borda

	if tela != esperado {
		t.Errorf("a tela vazia não bateu com o quadro esperado:\n%s", tela)
	}
}

// TestMontarDimensoes confere o tamanho da tela montada.
func TestMontarDimensoes(t *testing.T) {
	tela, err := Montar(novaMemoria())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	linhas := linhasDe(tela)
	if len(linhas) != Linhas+2 { // as linhas da tela mais as duas bordas
		t.Errorf("a tela tem %d linhas, esperado %d", len(linhas), Linhas+2)
	}
	for i, linha := range linhas {
		if len(linha) != Colunas+2 { // os caracteres mais as duas bordas laterais
			t.Errorf("linha %d tem %d caracteres, esperado %d", i, len(linha), Colunas+2)
		}
	}
}

// TestMontarQuadroConhecido escreve "HI" no começo e confere a primeira linha.
func TestMontarQuadroConhecido(t *testing.T) {
	mem := novaMemoria()
	mem.Write32(enderecoDe(0, 0), 'H')
	mem.Write32(enderecoDe(0, 1), 'I')

	tela, err := Montar(mem)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	esperado := "|HI" + strings.Repeat(" ", Colunas-2) + "|"
	if linhas := linhasDe(tela); linhas[1] != esperado {
		t.Errorf("primeira linha = %q, esperado %q", linhas[1], esperado)
	}
}

// TestMontarPosicao confere que cada célula da VRAM cai na linha e coluna certas.
func TestMontarPosicao(t *testing.T) {
	casos := []struct {
		nome          string
		linha, coluna int
		caractere     byte
	}{
		{"canto superior esquerdo", 0, 0, 'A'},
		{"canto superior direito", 0, Colunas - 1, 'B'},
		{"meio", 1, 2, 'X'},
		{"canto inferior esquerdo", Linhas - 1, 0, 'C'},
		{"canto inferior direito", Linhas - 1, Colunas - 1, 'D'},
	}

	for _, c := range casos {
		mem := novaMemoria()
		mem.Write32(enderecoDe(c.linha, c.coluna), uint32(c.caractere))

		tela, err := Montar(mem)
		if err != nil {
			t.Fatalf("%s: erro inesperado: %v", c.nome, err)
		}
		// a linha 0 da tela está em linhas[1] (linhas[0] é a borda),
		// e a coluna 0 está no índice 1 (o índice 0 é a borda lateral)
		obtido := linhasDe(tela)[c.linha+1][c.coluna+1]
		if obtido != c.caractere {
			t.Errorf("%s: apareceu %q, esperado %q", c.nome, obtido, c.caractere)
		}
	}
}

// TestMontarCaracteresNaoImprimiveis confere o que vira espaço e o que aparece.
func TestMontarCaracteresNaoImprimiveis(t *testing.T) {
	casos := []struct {
		nome     string
		palavra  uint32
		esperado byte
	}{
		{"zero (memória vazia)", 0, ' '},
		{"quebra de linha (10)", 10, ' '},
		{"31, logo antes do espaço", 31, ' '},
		{"32, o espaço", 32, ' '},
		{"A", 'A', 'A'},
		{"~ (126, o último imprimível)", 126, '~'},
		{"127 (DEL)", 127, ' '},
		{"200 (fora do ASCII)", 200, ' '},
		{"só o byte menor conta (0x141 = 'A')", 0x141, 'A'},
		{"byte menor zero (0x4100)", 0x4100, ' '},
	}

	for _, c := range casos {
		mem := novaMemoria()
		mem.Write32(enderecoDe(0, 0), c.palavra)

		tela, err := Montar(mem)
		if err != nil {
			t.Fatalf("%s: erro inesperado: %v", c.nome, err)
		}
		if obtido := linhasDe(tela)[1][1]; obtido != c.esperado {
			t.Errorf("%s: apareceu %q, esperado %q", c.nome, obtido, c.esperado)
		}
	}
}

// TestMontarErroDeLeitura confere que uma falha na memória vira erro.
func TestMontarErroDeLeitura(t *testing.T) {
	mem := novaMemoria()
	mem.falhaEm[enderecoDe(3, 5)] = true

	if _, err := Montar(mem); !errors.Is(err, errFalso) {
		t.Errorf("erro = %v, esperado a falha simulada", err)
	}
}

// TestDesenharErro confere que o Desenhar também devolve o erro.
func TestDesenharErro(t *testing.T) {
	mem := novaMemoria()
	mem.falhaEm[enderecoDe(0, 0)] = true

	if err := Desenhar(mem); !errors.Is(err, errFalso) {
		t.Errorf("erro = %v, esperado a falha simulada", err)
	}
}
