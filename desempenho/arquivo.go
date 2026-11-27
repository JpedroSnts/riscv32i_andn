package desempenho

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// LerPalavras lê um programa em texto: uma instrução por linha, em hexadecimal,
// com ou sem 0x. Aceita linhas em branco, comentários (// ou #) e uma vírgula
// no final da linha, para dar para colar de um código Go.
func LerPalavras(entrada io.Reader) ([]uint32, error) {
	var palavras []uint32
	leitor := bufio.NewScanner(entrada)
	numero := 0

	for leitor.Scan() {
		numero++
		texto := leitor.Text()
		if numero == 1 {
			texto = strings.TrimPrefix(texto, "\ufeff") // marcador que o Bloco de Notas pode colocar
		}

		// tira o comentário
		if i := strings.Index(texto, "//"); i >= 0 {
			texto = texto[:i]
		}
		if i := strings.Index(texto, "#"); i >= 0 {
			texto = texto[:i]
		}
		texto = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(texto), ","))
		if texto == "" {
			continue // linha em branco ou só de comentário
		}

		digitos := texto
		if strings.HasPrefix(texto, "0x") || strings.HasPrefix(texto, "0X") {
			digitos = texto[2:]
		}
		valor, err := strconv.ParseUint(digitos, 16, 32)
		if err != nil || len(digitos) == 0 || len(digitos) > 8 {
			return nil, fmt.Errorf("linha %d: %q não é uma instrução em hexadecimal de até 8 dígitos", numero, texto)
		}
		palavras = append(palavras, uint32(valor))
	}
	if err := leitor.Err(); err != nil {
		return nil, err
	}
	if len(palavras) == 0 {
		return nil, fmt.Errorf("o programa não tem nenhuma instrução")
	}
	return palavras, nil
}

// LerArquivo lê um programa de um arquivo. O nome do programa é o nome do arquivo sem a extensão. Não tem resultado esperado, então não é validado.
func LerArquivo(caminho string) (Programa, error) {
	arquivo, err := os.Open(caminho)
	if err != nil {
		return Programa{}, err
	}
	defer arquivo.Close()

	palavras, err := LerPalavras(arquivo)
	if err != nil {
		return Programa{}, fmt.Errorf("%s: %w", filepath.Base(caminho), err)
	}

	nome := strings.TrimSuffix(filepath.Base(caminho), filepath.Ext(caminho))
	return Programa{Nome: nome, Palavras: palavras}, nil
}

// EscreverPalavras escreve um programa no mesmo formato que o LerPalavras lê.
func EscreverPalavras(saida io.Writer, palavras []uint32) error {
	for _, palavra := range palavras {
		if _, err := fmt.Fprintf(saida, "0x%08X\n", palavra); err != nil {
			return err
		}
	}
	return nil
}
