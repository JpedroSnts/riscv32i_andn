package teclado

import (
	"fmt"
	"sync"
	"sync/atomic"

	"riscv/barramento"
)

// Endereços do teclado, dentro da região de periféricos do enunciado.
const (
	Base        = 0x9FC00     // início da região de periféricos
	Tamanho     = 0x400       // tamanho da região (até 0x9FFFF)
	RegStatus   = Base + 0x00 // leitura: bit 0 = há tecla pendente
	RegTecla    = Base + 0x04 // leitura: código da tecla; ler limpa a pendência
	RegControle = Base + 0x08 // bit 0 = interrupções habilitadas
)

// TeclaInterrupcao é a única tecla que gera interrupção.
const TeclaInterrupcao = 'i'

// Teclado é um periférico que guarda a última tecla de interrupção apertada.
type Teclado struct {
	mu         sync.Mutex // protege os campos abaixo (o terminal roda em paralelo)
	tecla      uint32     // código da última tecla de interrupção
	pendente   bool       // true se a tecla ainda não foi lida
	habilitada bool       // true se as interrupções estão ligadas

	// linha é uma cópia de (pendente && habilitada), para a CPU consultar a cada instrução sem precisar da trava. Só atualizar() a escreve.
	linha atomic.Bool
}

// Novo cria um teclado sem tecla pendente e com interrupções desligadas.
func Novo() *Teclado {
	return &Teclado{}
}

// atualizar recalcula a linha de interrupção. Precisa ser chamada, com a trava, sempre que pendente ou habilitada mudar.
func (t *Teclado) atualizar() {
	t.linha.Store(t.pendente && t.habilitada)
}

// Pressionar é chamado quando uma tecla é apertada.
// Só a tecla de interrupção é registrada; as outras são ignoradas.
func (t *Teclado) Pressionar(tecla byte) {
	if tecla != TeclaInterrupcao {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.tecla = uint32(tecla)
	t.pendente = true
	t.atualizar()
}

// Interrupcao é a linha de interrupção: true se há tecla pendente e as interrupções estão habilitadas. Não usa a trava: a CPU a consulta depois de cada instrução, e travar ali custava mais da metade do tempo.
func (t *Teclado) Interrupcao() bool {
	return t.linha.Load()
}

// Read32 lê um dos registradores do teclado.
func (t *Teclado) Read32(endereco uint32) (uint32, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch endereco {
	case RegStatus:
		return booleano(t.pendente), nil
	case RegTecla:
		t.pendente = false // ler a tecla limpa a pendência
		t.atualizar()
		return t.tecla, nil
	case RegControle:
		return booleano(t.habilitada), nil
	}
	return 0, fmt.Errorf("teclado: endereço %#x não é um registrador", endereco)
}

// Write32 escreve no registrador de controle; os outros são só de leitura.
func (t *Teclado) Write32(endereco uint32, valor uint32) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch endereco {
	case RegControle:
		t.habilitada = valor&1 == 1 // só o bit 0 conta
		t.atualizar()
		return nil
	case RegStatus, RegTecla:
		return fmt.Errorf("teclado: o registrador %#x é somente leitura", endereco)
	}
	return fmt.Errorf("teclado: endereço %#x não é um registrador", endereco)
}

// booleano converte true em 1 e false em 0.
func booleano(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

// O compilador confere que o Teclado é um Barramento (pode ser ligado ao mapa) e uma fonte de interrupção (pode ser registrado no barramento do sistema).
var _ barramento.Barramento = (*Teclado)(nil)
var _ barramento.FonteInterrupcao = (*Teclado)(nil)
