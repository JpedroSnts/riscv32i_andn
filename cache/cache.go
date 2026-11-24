package cache

import "riscv/barramento"

const (
	NumLinhas   = 256 // linhas da cache
	LimiteCache = 0x80000 // só endereços abaixo disto passam pela cache (RAM principal)
)

// linha é uma posição da cache: guarda uma palavra e de onde ela veio.
type linha struct {
	valida bool // false = a linha ainda não foi preenchida
	tag    uint32 // parte alta do endereço, diz quem é o dono da linha
	dado   uint32 // a palavra guardada
}

// Cache fica entre a CPU e o barramento do sistema.
// Mapeamento direto: cada endereço só pode ocupar uma linha.
type Cache struct {
	linhas  [NumLinhas]linha
	destino barramento.Barramento // quem está do outro lado (o barramento do sistema)

	Hits int
	Misses int 
}

// NovaCache cria uma cache vazia ligada a um destino.
func NovaCache(destino barramento.Barramento) *Cache {
	return &Cache{destino: destino}
}

// indiceDe devolve em qual linha o endereço deve ficar (bits 9:2).
func indiceDe(endereco uint32) uint32 {
	return (endereco / 4) % NumLinhas
}

// tagDe devolve a tag do endereço (bits 31:10).
func tagDe(endereco uint32) uint32 {
	return endereco / (4 * NumLinhas)
}

// Read32 devolve a palavra do endereço, usando a cache quando possível.
func (c *Cache) Read32(endereco uint32) (uint32, error) {
	// VRAM e periféricos não passam pela cache, nem endereço desalinhado (o destino devolve o erro de alinhamento).
	if endereco >= LimiteCache || endereco%4 != 0 {
		return c.destino.Read32(endereco)
	}

	// l é a linha onde este endereço precisa ficar (um ponteiro, para poder alterá-la)
	l := &c.linhas[indiceDe(endereco)]
	tag := tagDe(endereco)

	//cache hit
	if l.valida && l.tag == tag {
		c.Hits++
		return l.dado, nil
	}

	//cache miss
	dado, err := c.destino.Read32(endereco)
	if err != nil {
		return 0, err
	}
	c.Misses++

	//guarda na linha, substituindo quem estava lá
	*l = linha{valida: true, tag: tag, dado: dado}
	return dado, nil
}

// Write32 grava o valor na memória e guarda uma cópia na cache (write-through).
func (c *Cache) Write32(endereco uint32, valor uint32) error {
	// VRAM e periféricos não passam pela cache, nem endereço desalinhado.
	if endereco >= LimiteCache || endereco%4 != 0 {
		return c.destino.Write32(endereco, valor)
	}

	// a memória é atualizada primeiro; se falhar, a cache não muda
	if err := c.destino.Write32(endereco, valor); err != nil {
		return err
	}

	l := &c.linhas[indiceDe(endereco)]
	tag := tagDe(endereco)

	// hit ou miss
	if l.valida && l.tag == tag {
		c.Hits++
	} else {
		c.Misses++
	}

	// guarda a cópia, substituindo quem estava lá
	*l = linha{valida: true, tag: tag, dado: valor}
	return nil
}

// O compilador confere se a Cache é um Barramento. Se faltar algum método, o erro aparece aqui.
var _ barramento.Barramento = (*Cache)(nil)