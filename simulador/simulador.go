package simulador

import (
	"fmt"
	"riscv/barramento"
	"riscv/cache"
	"riscv/cpu"
	"riscv/mem"
)

// Simulador junta as peças: mapa de memória, barramento e CPU.
type Simulador struct {
	Mapa       *mem.Mapa           // memória e dispositivos
	Barramento *barramento.Sistema // linhas de endereço, dados e controle
	Cache      *cache.Cache        // fica entre a CPU e o barramento
	CPU        *cpu.CPU
}

// Novo monta a máquina e carrega o programa a partir do endereço 0.
func Novo(programa []uint32) (*Simulador, error) {
	mapa, err := mem.NovoMapaPadrao()
	if err != nil {
		return nil, err
	}
	if err := mapa.Load(0, programa); err != nil {
		return nil, err
	}

	// o barramento fica entre a cache e o mapa de memória
	sistema := barramento.NovoSistema(mapa)

	// a cache fica entre a CPU e o barramento
	c := cache.NovaCache(sistema)

	return &Simulador{Mapa: mapa, Barramento: sistema, Cache: c, CPU: cpu.New(c)}, nil
}

// Executar roda a CPU até o programa terminar (ebreak) ou chegar a limitePassos.
// A cada "intervalo" instruções chama aoIntervalo, que pode ser nil (o intervalo precisa ser maior que 0 para valer).
// Devolve quantas instruções foram executadas.
func (s *Simulador) Executar(limitePassos, intervalo int, aoIntervalo func(passos int)) (int, error) {
	passos := 0
	for !s.CPU.Parou && passos < limitePassos {
		if err := s.CPU.Step(); err != nil {
			return passos, err
		}
		passos++

		if aoIntervalo != nil && intervalo > 0 && passos%intervalo == 0 {
			aoIntervalo(passos)
		}
	}

	if !s.CPU.Parou {
		return passos, fmt.Errorf("limite de %d passos atingido: o programa não terminou", limitePassos)
	}
	return passos, nil
}