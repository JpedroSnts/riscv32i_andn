package barramento

import (
	"errors"
	"testing"
)

// destinoFalso é uma memória mínima que pode ser configurada para falhar.
type destinoFalso struct {
	palavras map[uint32]uint32
	falha    bool
}

var errDestino = errors.New("falha simulada")

func (d *destinoFalso) Read32(endereco uint32) (uint32, error) {
	if d.falha {
		return 0, errDestino
	}
	return d.palavras[endereco], nil
}

func (d *destinoFalso) Write32(endereco uint32, valor uint32) error {
	if d.falha {
		return errDestino
	}
	d.palavras[endereco] = valor
	return nil
}

// espiao anota o que viu nas linhas do barramento no instante do acesso.
type espiao struct {
	sistema       *Sistema
	controleVisto Controle
	enderecoVisto uint32
	dadosVistos   uint32
}

func (e *espiao) Read32(endereco uint32) (uint32, error) {
	e.controleVisto = e.sistema.Controle
	e.enderecoVisto = e.sistema.Endereco
	return 5, nil
}

func (e *espiao) Write32(endereco uint32, valor uint32) error {
	e.controleVisto = e.sistema.Controle
	e.enderecoVisto = e.sistema.Endereco
	e.dadosVistos = e.sistema.Dados
	return nil
}

// TestSistemaLeitura confere o resultado e o estado das linhas após uma leitura.
func TestSistemaLeitura(t *testing.T) {
	destino := &destinoFalso{palavras: map[uint32]uint32{0x100: 42}}
	s := NovoSistema(destino)

	valor, err := s.Read32(0x100)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if valor != 42 {
		t.Errorf("valor = %d, esperado 42", valor)
	}
	if s.Endereco != 0x100 {
		t.Errorf("Endereco = %#x, esperado 0x100", s.Endereco)
	}
	if s.Dados != 42 {
		t.Errorf("Dados = %d, esperado 42", s.Dados)
	}
	if s.Controle != Ocioso {
		t.Errorf("Controle = %v, esperado ocioso depois do acesso", s.Controle)
	}
}

// TestSistemaEscrita confere o estado das linhas e o destino após uma escrita.
func TestSistemaEscrita(t *testing.T) {
	destino := &destinoFalso{palavras: map[uint32]uint32{}}
	s := NovoSistema(destino)

	if err := s.Write32(0x104, 7); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if destino.palavras[0x104] != 7 {
		t.Errorf("o destino guardou %d, esperado 7", destino.palavras[0x104])
	}
	if s.Endereco != 0x104 {
		t.Errorf("Endereco = %#x, esperado 0x104", s.Endereco)
	}
	if s.Dados != 7 {
		t.Errorf("Dados = %d, esperado 7", s.Dados)
	}
	if s.Controle != Ocioso {
		t.Errorf("Controle = %v, esperado ocioso depois do acesso", s.Controle)
	}
}

// TestSistemaControleDuranteOAcesso confere que a linha de controle indica leitura ou escrita ENQUANTO o destino atende o pedido.
func TestSistemaControleDuranteOAcesso(t *testing.T) {
	esp := &espiao{}
	s := NovoSistema(esp)
	esp.sistema = s

	s.Read32(0x200)
	if esp.controleVisto != Leitura {
		t.Errorf("durante a leitura, Controle = %v, esperado leitura", esp.controleVisto)
	}
	if esp.enderecoVisto != 0x200 {
		t.Errorf("durante a leitura, Endereco = %#x, esperado 0x200", esp.enderecoVisto)
	}

	s.Write32(0x204, 9)
	if esp.controleVisto != Escrita {
		t.Errorf("durante a escrita, Controle = %v, esperado escrita", esp.controleVisto)
	}
	if esp.enderecoVisto != 0x204 {
		t.Errorf("durante a escrita, Endereco = %#x, esperado 0x204", esp.enderecoVisto)
	}
	// na escrita, o valor já precisa estar nas linhas de dados
	if esp.dadosVistos != 9 {
		t.Errorf("durante a escrita, Dados = %d, esperado 9", esp.dadosVistos)
	}
}

// TestSistemaErroDoDestino confere que a falha passa adiante e que uma leitura com erro não deixa lixo nas linhas de dados.
func TestSistemaErroDoDestino(t *testing.T) {
	destino := &destinoFalso{palavras: map[uint32]uint32{0x100: 42}}
	s := NovoSistema(destino)

	// uma leitura boa, para as linhas terem um valor conhecido
	if _, err := s.Read32(0x100); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	destino.falha = true

	if _, err := s.Read32(0x100); !errors.Is(err, errDestino) {
		t.Errorf("leitura: erro = %v, esperado a falha simulada", err)
	}
	if s.Dados != 42 {
		t.Errorf("Dados = %d, uma leitura com erro não deveria alterá-lo (era 42)", s.Dados)
	}
	if s.Controle != Ocioso {
		t.Errorf("Controle = %v, esperado ocioso mesmo depois do erro", s.Controle)
	}

	if err := s.Write32(0x100, 1); !errors.Is(err, errDestino) {
		t.Errorf("escrita: erro = %v, esperado a falha simulada", err)
	}
	if s.Controle != Ocioso {
		t.Errorf("Controle = %v, esperado ocioso mesmo depois do erro", s.Controle)
	}
}

// TestControleTexto confere os nomes que aparecem ao imprimir o estado.
func TestControleTexto(t *testing.T) {
	casos := map[Controle]string{
		Ocioso:  "ocioso",
		Leitura: "leitura",
		Escrita: "escrita",
	}
	for controle, esperado := range casos {
		if obtido := controle.String(); obtido != esperado {
			t.Errorf("String() = %q, esperado %q", obtido, esperado)
		}
	}
}