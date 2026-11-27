package teclado

import (
	"bufio"
	"io"
)

// LerDe lê a entrada byte a byte e chama Pressionar para cada um, até a entrada acabar ou dar erro. É feita para rodar numa goroutine, em paralelo com a CPU: no programa, a entrada é o terminal.
func (t *Teclado) LerDe(entrada io.Reader) {
	leitor := bufio.NewReader(entrada)
	for {
		tecla, err := leitor.ReadByte()
		if err != nil {
			return // a entrada fechou (ou falhou): não há mais teclas
		}
		t.Pressionar(tecla) // as teclas que não são a de interrupção são ignoradas lá dentro
	}
}
