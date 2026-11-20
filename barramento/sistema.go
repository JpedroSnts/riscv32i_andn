package barramento


type Controle int
const (
	Ocioso  Controle = 0 // nenhum acesso em andamento
	Leitura Controle = 1 // a CPU está lendo da memória
	Escrita Controle = 2 // a CPU está escrevendo na memória
)

// String devolve o nome do estado
func (c Controle) String() string {
	switch c {
	case Leitura:
		return "leitura"
	case Escrita:
		return "escrita"
	}
	return "ocioso"
}

// Sistema é o barramento do computador
// Fica entre a CPU e a memória/E/S
type Sistema struct {
	Endereco uint32   // Onde ler ou escrever
	Dados    uint32   // O valor que trafega
	Controle Controle // Leitura, escrita ou ocioso

	destino Barramento 
}

// NovoSistema cria o barramento ligado a um destino.
func NovoSistema(destino Barramento) *Sistema {
	return &Sistema{destino: destino}
}

// Read32 faz uma leitura passando pelas três linhas do barramento.
func (s *Sistema) Read32(endereco uint32) (uint32, error) {
	s.Endereco = endereco // O endereço vai para as linhas de endereço
	s.Controle = Leitura  // A linha de controle indica leitura

	dado, err := s.destino.Read32(endereco)
	if err == nil {
		s.Dados = dado // O valor lido aparece nas linhas de dados
	}

	s.Controle = Ocioso // Acesso terminado
	return dado, err
}

// Write32 faz uma escrita passando pelas três linhas do barramento.
func (s *Sistema) Write32(endereco uint32, valor uint32) error {
	s.Endereco = endereco // O endereço vai para as linhas de endereço
	s.Dados = valor       // O valor vai para as linhas de dados
	s.Controle = Escrita  // A linha de controle indica escrita

	// O destino (mapa de memória) grava o valor
	err := s.destino.Write32(endereco, valor)

	s.Controle = Ocioso // 5. acesso terminado
	return err
}