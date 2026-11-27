package mem

import "fmt"

// RAM é uma memória de palavras de 32 bits (uma "linha" por palavra).
// Serve para a RAM principal e também para a VRAM, mudando só a base.
type RAM struct {
	base     uint32   // primeiro endereço que esta memória atende
	palavras []uint32 // cada posição é uma palavra de 32 bits
}

// NovaRAM cria uma memória que começa em base e tem tamanho bytes.
func NovaRAM(base, tamanho uint32) *RAM {
	return &RAM{base: base, palavras: make([]uint32, tamanho/4)}
}

// indice converte um endereço em posição da lista.
// Devolve erro se o endereço estiver desalinhado ou fora da memória.
func (m *RAM) indice(endereco uint32) (uint32, error) {
	if endereco%4 != 0 {
		return 0, fmt.Errorf("endereço %#x desalinhado (precisa ser múltiplo de 4)", endereco)
	}
	if endereco < m.base || endereco-m.base >= uint32(len(m.palavras))*4 {
		return 0, fmt.Errorf("endereço %#x fora da memória", endereco)
	}
	return (endereco - m.base) / 4, nil
}

// Read32 devolve a palavra guardada no endereço.
func (m *RAM) Read32(endereco uint32) (uint32, error) {
	posicao, err := m.indice(endereco)
	if err != nil {
		return 0, err
	}
	return m.palavras[posicao], nil
}

// Write32 grava o valor no endereço.
func (m *RAM) Write32(endereco uint32, valor uint32) error {
	posicao, err := m.indice(endereco)
	if err != nil {
		return err
	}
	m.palavras[posicao] = valor
	return nil
}

func (m *RAM) Load(endereco uint32, programa []uint32) error {
	for n, instrucao := range programa {
		if err := m.Write32(endereco+uint32(n)*4, instrucao); err != nil {
			return err
		}
	}
	return nil
}
