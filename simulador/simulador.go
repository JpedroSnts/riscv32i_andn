package simulador

import (
	"fmt"
	"riscv/barramento"
	"riscv/cache"
	"riscv/cpu"
	"riscv/mem"
	"riscv/teclado"
)

// Simulador junta as peças: mapa de memória, barramento e CPU.
type Simulador struct {
	Mapa       *mem.Mapa           // memória e dispositivos
	Barramento *barramento.Sistema // linhas de endereço, dados, controle e interrupção
	Cache      *cache.Cache        // fica entre a CPU e o barramento
	Teclado    *teclado.Teclado    // o periférico que gera a interrupção
	CPU        *cpu.CPU
}

// Novo monta a máquina e carrega o programa a partir do endereço 0.
func Novo(programa []uint32) (*Simulador, error) {
	// o teclado ocupa a região de periféricos do mapa
	tec := teclado.Novo()

	mapa, err := mem.NovoMapaCom(tec)
	if err != nil {
		return nil, err
	}
	if err := mapa.Load(0, programa); err != nil {
		return nil, err
	}

	// o barramento fica entre a cache e o mapa de memória, e o teclado pode pedir interrupção por ele
	sistema := barramento.NovoSistema(mapa)
	sistema.AdicionarFonte(tec)

	// a cache fica entre a CPU e o barramento
	c := cache.NovaCache(sistema)

	processador := cpu.New(c)
	processador.LinhaInterrupcao = sistema // a CPU consulta a linha de interrupção do barramento

	return &Simulador{Mapa: mapa, Barramento: sistema, Cache: c, Teclado: tec, CPU: processador}, nil
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
