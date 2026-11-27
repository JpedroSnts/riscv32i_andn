package video

import (
	"fmt"
	"riscv/barramento"
	"strings"
)

// Formato da tela de texto
const (
	Base    = 0x80000 // início da VRAM
	Colunas = 40      // caracteres por linha
	Linhas  = 10      // quantidade de linhas
)

// Montar lê a VRAM e devolve o texto da tela, com borda, pronto para imprimir.
// memoria é quem permite ler a VRAM (o mapa de memória).
func Montar(memoria barramento.Barramento) (string, error) {
	var tela strings.Builder // monta o texto todo antes de devolver

	borda := "+" + strings.Repeat("-", Colunas) + "+\n"
	tela.WriteString(borda)

	for linha := 0; linha < Linhas; linha++ {
		tela.WriteString("|")
		for coluna := 0; coluna < Colunas; coluna++ {
			// cada caractere ocupa uma palavra de 4 bytes
			endereco := uint32(Base + (linha*Colunas+coluna)*4)
			palavra, err := memoria.Read32(endereco)
			if err != nil {
				return "", err
			}

			// só o byte menor é o caractere; o que não for imprimível vira espaço
			caractere := byte(palavra & 0xFF)
			if caractere < 32 || caractere > 126 {
				caractere = ' '
			}
			tela.WriteByte(caractere)
		}
		tela.WriteString("|\n")
	}

	tela.WriteString(borda)
	return tela.String(), nil
}

// Desenhar monta a tela e a imprime no terminal.
func Desenhar(memoria barramento.Barramento) error {
	tela, err := Montar(memoria)
	if err != nil {
		return err
	}
	fmt.Print(tela)
	return nil
}
