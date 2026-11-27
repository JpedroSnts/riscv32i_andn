package teclado

import (
	"io"
	"strings"
	"testing"
	"time"
)

// TestLerDeTeclaI confere que um 'i' na entrada vira pendência.
func TestLerDeTeclaI(t *testing.T) {
	k := Novo()
	k.LerDe(strings.NewReader("i"))

	if ler(t, k, RegStatus) != 1 {
		t.Error("STATUS deveria ser 1 depois de ler um 'i'")
	}
}

// TestLerDeComEnter confere o que o terminal entrega de verdade: a tecla e o Enter.
func TestLerDeComEnter(t *testing.T) {
	k := Novo()
	k.LerDe(strings.NewReader("i\r\n")) // o Windows manda \r\n

	if ler(t, k, RegStatus) != 1 {
		t.Error("o 'i' seguido de Enter deveria gerar a pendência")
	}
}

// TestLerDeIgnoraOutras confere que só o 'i' minúsculo conta.
func TestLerDeIgnoraOutras(t *testing.T) {
	k := Novo()
	k.LerDe(strings.NewReader("abc\r\n xyz I\n"))

	if ler(t, k, RegStatus) != 0 {
		t.Error("nenhuma dessas teclas deveria gerar pendência")
	}
}

// TestLerDeVariasTeclas confere que várias teclas seguidas viram uma pendência só.
func TestLerDeVariasTeclas(t *testing.T) {
	k := Novo()
	k.LerDe(strings.NewReader("iii"))

	if ler(t, k, RegStatus) != 1 {
		t.Fatal("STATUS deveria ser 1")
	}
	ler(t, k, RegTecla) // uma leitura limpa tudo
	if ler(t, k, RegStatus) != 0 {
		t.Error("uma leitura da TECLA deveria limpar a pendência")
	}
}

// TestLerDeEntradaVazia confere que uma entrada sem nada termina sem problema.
func TestLerDeEntradaVazia(t *testing.T) {
	k := Novo()
	k.LerDe(strings.NewReader(""))

	if ler(t, k, RegStatus) != 0 {
		t.Error("STATUS deveria continuar 0")
	}
}

// TestLerDeEmParalelo confere o uso de verdade: o leitor roda numa goroutine
// enquanto outro código consulta a linha de interrupção.
func TestLerDeEmParalelo(t *testing.T) {
	k := Novo()
	escrever(t, k, RegControle, 1)

	leitura, escrita := io.Pipe() // uma entrada que o teste controla
	feito := make(chan struct{})
	go func() {
		k.LerDe(leitura)
		close(feito) // avisa que o leitor terminou
	}()

	if k.Interrupcao() {
		t.Fatal("ainda não foi digitado nada: não deveria haver interrupção")
	}

	escrita.Write([]byte("i\r\n")) // simula o usuário digitando

	// espera a goroutine processar a tecla
	limite := time.After(2 * time.Second)
	for !k.Interrupcao() {
		select {
		case <-limite:
			t.Fatal("a tecla não chegou ao teclado a tempo")
		default:
			time.Sleep(time.Millisecond)
		}
	}

	// fechando a entrada, o leitor tem que terminar
	escrita.Close()
	select {
	case <-feito:
	case <-time.After(2 * time.Second):
		t.Error("o LerDe deveria terminar quando a entrada fecha")
	}
}
