package mem

import (
	"fmt"
	"riscv/barramento"
)

// regiao é um pedaço do mapa de memória.
type regiao struct {
	nome        string  // nome só para mensagens de erro (ex.: "VRAM")
	base        uint32  // primeiro endereço da região
	tamanho     uint32  // quantos bytes a região ocupa
	dispositivo barramento.Barramento // quem atende os acessos; nil = reservada (sem dono)
}

// Mapa direciona cada endereço para a região que é dona dele.
// Ele também é um Barramento, então a CPU o usa normalmente.
type Mapa struct {
	regioes []regiao
}

// Adicionar registra uma região. Recusa se ela se sobrepuser a outra.
func (m *Mapa) adicionar(nome string, base, tamanho uint32, dispositivo barramento.Barramento) error {
	fim := base + tamanho // primeiro endereço depois da região
	for _, r := range m.regioes {
		if base < r.base + r.tamanho && r.base < fim {
			return fmt.Errorf("região %s se sobrepõe a %s", nome, r.nome)
		}
	}
	m.regioes = append(m.regioes, regiao{nome, base, tamanho, dispositivo})

	return nil
}

// buscar acha a região que contém o endereço.
func (m *Mapa) buscar(endereco uint32) (*regiao, error) {
	for i := range m.regioes {
		r := &m.regioes[i]
	if endereco >= r.base && endereco-r.base < r.tamanho {
			return r, nil
		}
	}
	return nil, fmt.Errorf("endereço %#x não está mapeado", endereco)
}	

// Read32 encaminha a leitura para o dono do endereço.
func (m *Mapa) Read32(endereco uint32) (uint32, error) {
	r, err := m.buscar(endereco)
	if err != nil {
		return 0, err
	}
	if r.dispositivo == nil {
		return 0, fmt.Errorf("endereço %#x está na região reservada (%s)", endereco, r.nome)
	}
	return r.dispositivo.Read32(endereco)
}

// Write32 encaminha a escrita para o dono do endereço.
func (m *Mapa) Write32(endereco uint32, valor uint32) error {
	r, err := m.buscar(endereco)
	if err != nil {
		return err
	}
	if r.dispositivo == nil {
		return fmt.Errorf("endereço %#x está na região reservada (%s)", endereco, r.nome)
	}
	return r.dispositivo.Write32(endereco, valor)
}

// Carregar coloca um programa a partir do endereço, uma instrução a cada 4 bytes.
func (m *Mapa) Carregar(endereco uint32, programa []uint32) error {
	for n, instrucao := range programa {
		if err := m.Write32(endereco+uint32(n)*4, instrucao); err != nil {
			return err
		}
	}
	return nil
}

// NovoMapaPadrao monta o mapa de memória do enunciado.
func NovoMapaPadrao() (*Mapa, error) {
	m := &Mapa{}

	// Tabela com as quatro regiões do PDF.
	// Cada linha é: nome, base, tamanho, dispositivo.
	tabela := []regiao{
		{"RAM principal", 0x00000, 0x80000, NovaRAM(0x00000, 0x80000)},
		{"VRAM", 0x80000, 0x10000, NovaRAM(0x80000, 0x10000)},
		{"Reservada", 0x90000, 0xFC00, nil}, // nil = sem dono, acesso dá erro
		{"Periféricos", 0x9FC00, 0x400, NovaRAM(0x9FC00, 0x400)},
	}

	// r é cada linha da tabela, registrada no mapa uma por uma.
	for _, r := range tabela {
		if err := m.adicionar(r.nome, r.base, r.tamanho, r.dispositivo); err != nil {
			return nil, err
		}
	}
	return m, nil
}

