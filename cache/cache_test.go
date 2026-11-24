package cache

import (
	"errors"
	"testing"
)

// memoriaFalsa guarda palavras num mapa, conta os acessos que recebe e pode ser configurada para falhar.
type memoriaFalsa struct {
	palavras map[uint32]uint32
	leituras int  // quantas vezes o Read32 foi chamado
	escritas int  // quantas vezes o Write32 foi chamado
	falha    bool // se true, qualquer acesso dá erro
}

var errFalso = errors.New("falha simulada")

func (m *memoriaFalsa) Read32(endereco uint32) (uint32, error) {
	m.leituras++
	if m.falha {
		return 0, errFalso
	}
	return m.palavras[endereco], nil
}

func (m *memoriaFalsa) Write32(endereco uint32, valor uint32) error {
	m.escritas++
	if m.falha {
		return errFalso
	}
	m.palavras[endereco] = valor
	return nil
}

// nova cria uma cache ligada a uma memória falsa nova.
func nova() (*Cache, *memoriaFalsa) {
	mem := &memoriaFalsa{palavras: map[uint32]uint32{}}
	return NovaCache(mem), mem
}

// confere compara os contadores da cache com o esperado.
func confere(t *testing.T, c *Cache, hits, misses int) {
	t.Helper()
	if c.Hits != hits || c.Misses != misses {
		t.Errorf("hits/misses = %d/%d, esperado %d/%d", c.Hits, c.Misses, hits, misses)
	}
}

// TestIndiceETag confere a divisão do endereço em índice e tag.
func TestIndiceETag(t *testing.T) {
	casos := []struct {
		endereco uint32
		indice   uint32
		tag      uint32
	}{
		{0x000, 0, 0},
		{0x004, 1, 0},
		{0x100, 64, 0},
		{0x3FC, 255, 0},    // última linha da primeira volta
		{0x400, 0, 1},      // 256 palavras depois: volta à linha 0, tag nova
		{0x500, 64, 1},     // mesma linha que 0x100, tag diferente
		{0x7FFFC, 255, 511}, // último endereço da RAM
	}

	for _, c := range casos {
		if obtido := indiceDe(c.endereco); obtido != c.indice {
			t.Errorf("indiceDe(%#x) = %d, esperado %d", c.endereco, obtido, c.indice)
		}
		if obtido := tagDe(c.endereco); obtido != c.tag {
			t.Errorf("tagDe(%#x) = %d, esperado %d", c.endereco, obtido, c.tag)
		}
	}
}

// TestPrimeiroMissDepoisHit confere que a segunda leitura já vem da cache.
func TestPrimeiroMissDepoisHit(t *testing.T) {
	c, mem := nova()
	mem.palavras[0x100] = 42

	valor, err := c.Read32(0x100)
	if err != nil || valor != 42 {
		t.Fatalf("1ª leitura: valor = %d, erro = %v", valor, err)
	}
	confere(t, c, 0, 1) // linha vazia: miss
	if mem.leituras != 1 {
		t.Errorf("a memória recebeu %d leituras, esperado 1", mem.leituras)
	}

	valor, err = c.Read32(0x100)
	if err != nil || valor != 42 {
		t.Fatalf("2ª leitura: valor = %d, erro = %v", valor, err)
	}
	confere(t, c, 1, 1) // agora é hit
	if mem.leituras != 1 {
		t.Errorf("o hit não deveria chegar à memória: leituras = %d", mem.leituras)
	}
}

// TestEnderecosVizinhosConvivem confere que linhas diferentes não se atrapalham.
func TestEnderecosVizinhosConvivem(t *testing.T) {
	c, mem := nova()
	mem.palavras[0x100] = 1
	mem.palavras[0x104] = 2

	c.Read32(0x100) // miss
	c.Read32(0x104) // miss
	confere(t, c, 0, 2)

	// os dois continuam na cache
	if v, _ := c.Read32(0x100); v != 1 {
		t.Errorf("0x100 = %d, esperado 1", v)
	}
	if v, _ := c.Read32(0x104); v != 2 {
		t.Errorf("0x104 = %d, esperado 2", v)
	}
	confere(t, c, 2, 2)
}

// TestConflitoNaMesmaLinha confere que dois endereços com o mesmo índice se expulsam, e que o valor devolvido continua certo.
func TestConflitoNaMesmaLinha(t *testing.T) {
	c, mem := nova()
	mem.palavras[0x100] = 1
	mem.palavras[0x500] = 2 // mesmo índice (64), tag diferente

	passos := []struct {
		endereco uint32
		esperado uint32
	}{
		{0x100, 1}, // miss
		{0x500, 2}, // miss: expulsa o 0x100
		{0x100, 1}, // miss de novo: foi expulso
		{0x100, 1}, // hit: voltou para a cache
	}
	for _, p := range passos {
		if v, err := c.Read32(p.endereco); err != nil || v != p.esperado {
			t.Errorf("Read32(%#x) = %d, %v; esperado %d", p.endereco, v, err, p.esperado)
		}
	}
	confere(t, c, 1, 3)
}

// TestWriteThrough confere que a escrita vai para a memória e preenche a cache.
func TestWriteThrough(t *testing.T) {
	c, mem := nova()

	if err := c.Write32(0x100, 7); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if mem.palavras[0x100] != 7 {
		t.Errorf("a memória guardou %d, esperado 7 (write-through)", mem.palavras[0x100])
	}
	confere(t, c, 0, 1) // linha vazia: miss

	// a leitura seguinte já vem da cache, sem ir à memória
	if v, _ := c.Read32(0x100); v != 7 {
		t.Errorf("leitura = %d, esperado 7", v)
	}
	if mem.leituras != 0 {
		t.Errorf("a leitura não deveria chegar à memória: leituras = %d", mem.leituras)
	}
	confere(t, c, 1, 1)

	// escrever de novo no mesmo endereço é hit, e a memória recebe de novo
	c.Write32(0x100, 8)
	confere(t, c, 2, 1)
	if mem.escritas != 2 {
		t.Errorf("a memória recebeu %d escritas, esperado 2", mem.escritas)
	}
}

// TestEscritaAtualizaALinha confere que a cache nunca devolve um valor antigo.
func TestEscritaAtualizaALinha(t *testing.T) {
	c, mem := nova()
	mem.palavras[0x100] = 1

	c.Read32(0x100)     // traz o 1 para a cache
	c.Write32(0x100, 9) // atualiza memória e cache

	if v, _ := c.Read32(0x100); v != 9 {
		t.Errorf("leitura = %d, esperado 9 (valor novo)", v)
	}
	if mem.palavras[0x100] != 9 {
		t.Errorf("memória = %d, esperado 9", mem.palavras[0x100])
	}
}

// TestEscritaEmOutraTagSubstituiALinha confere o conflito também na escrita.
func TestEscritaEmOutraTagSubstituiALinha(t *testing.T) {
	c, mem := nova()
	mem.palavras[0x100] = 1

	c.Read32(0x100)      // miss: linha 64 guarda 0x100
	c.Write32(0x500, 5)  // mesma linha, tag diferente: miss, expulsa o 0x100
	confere(t, c, 0, 2)

	// 0x100 foi expulso: a leitura é miss e busca na memória
	if v, _ := c.Read32(0x100); v != 1 {
		t.Errorf("0x100 = %d, esperado 1", v)
	}
	confere(t, c, 0, 3)
}

// TestForaDaCache confere que VRAM e periféricos passam direto, sem contar.
func TestForaDaCache(t *testing.T) {
	enderecos := []uint32{
		0x80000, // primeiro da VRAM (LimiteCache)
		0x8FFFC, // último da VRAM
		0x9FC00, // periféricos
	}

	for _, e := range enderecos {
		c, mem := nova()
		mem.palavras[e] = 3

		// duas leituras: as duas precisam chegar à memória
		c.Read32(e)
		c.Read32(e)
		if mem.leituras != 2 {
			t.Errorf("%#x: a memória recebeu %d leituras, esperado 2", e, mem.leituras)
		}

		// e a escrita também chega, sem preencher a cache
		c.Write32(e, 4)
		if mem.escritas != 1 || mem.palavras[e] != 4 {
			t.Errorf("%#x: escrita não chegou à memória", e)
		}
		confere(t, c, 0, 0)
	}
}

// TestUltimoEnderecoCacheavel confere o limite: 0x7FFFC entra na cache, 0x80000 não.
func TestUltimoEnderecoCacheavel(t *testing.T) {
	c, mem := nova()
	mem.palavras[0x7FFFC] = 8

	c.Read32(0x7FFFC)
	c.Read32(0x7FFFC)
	confere(t, c, 1, 1)
	if mem.leituras != 1 {
		t.Errorf("leituras = %d, esperado 1", mem.leituras)
	}
}

// TestEnderecoDesalinhado confere que passa direto, sem contar e sem preencher.
func TestEnderecoDesalinhado(t *testing.T) {
	c, mem := nova()

	c.Read32(0x101)
	c.Read32(0x101)
	if mem.leituras != 2 {
		t.Errorf("a memória recebeu %d leituras, esperado 2", mem.leituras)
	}
	c.Write32(0x101, 1)
	if mem.escritas != 1 {
		t.Errorf("a memória recebeu %d escritas, esperado 1", mem.escritas)
	}
	confere(t, c, 0, 0)
}

// TestFalhaNaLeitura confere que uma leitura que falhou não suja a cache.
func TestFalhaNaLeitura(t *testing.T) {
	c, mem := nova()

	mem.falha = true
	if _, err := c.Read32(0x100); !errors.Is(err, errFalso) {
		t.Errorf("erro = %v, esperado a falha simulada", err)
	}
	confere(t, c, 0, 0) // nada foi trazido, então não conta miss

	// com a memória boa, a primeira leitura tem que ser miss (a linha não foi preenchida)
	mem.falha = false
	mem.palavras[0x100] = 5
	if v, _ := c.Read32(0x100); v != 5 {
		t.Errorf("valor = %d, esperado 5", v)
	}
	confere(t, c, 0, 1)
}

// TestFalhaNaEscrita confere que uma escrita que falhou não altera a cache.
func TestFalhaNaEscrita(t *testing.T) {
	c, mem := nova()
	mem.palavras[0x100] = 1
	c.Read32(0x100) // a linha passa a guardar 1

	mem.falha = true
	if err := c.Write32(0x100, 9); !errors.Is(err, errFalso) {
		t.Errorf("erro = %v, esperado a falha simulada", err)
	}
	confere(t, c, 0, 1) // a escrita que falhou não conta

	// a cache continua com o valor antigo, que ainda é o da memória
	mem.falha = false
	if v, _ := c.Read32(0x100); v != 1 {
		t.Errorf("valor = %d, esperado 1 (a escrita falhou)", v)
	}
	confere(t, c, 1, 1)
}