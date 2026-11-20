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

// Lê a VRAM e imprime no terminal os caracteres ASCII dela.
func Mostrar(memoria barramento.Barramento) error {
	var tela strings.Builder // monta o texto todo antes de imprimir

	borda := "+" + strings.Repeat("-", Colunas) + "+\n"
	tela.WriteString(borda)

	for linha := 0; linha < Linhas; linha++ {
		tela.WriteString("|")
		for coluna := 0; coluna < Colunas; coluna++ {
			// cada caractere ocupa uma palavra de 4 bytes
			endereco := uint32(Base + (linha*Colunas+coluna)*4)
			palavra, err := memoria.Read32(endereco)
			if err != nil {
				return err
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
	fmt.Print(tela.String())
	return nil
}