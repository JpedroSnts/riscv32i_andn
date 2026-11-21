package main

import (
	"fmt"
	"riscv/programas"
	"riscv/simulador"
	"riscv/video"
)

func main() {
	sim, err := simulador.Novo(programas.Alfabeto)
	if err != nil {
		fmt.Println("erro ao montar o simulador:", err)
		return
	}

	const intervaloTela = 20   // a VRAM é mostrada a cada 20 instruções
	const limitePassos = 10000 // trava de segurança contra laço infinito

	passos, err := sim.Executar(limitePassos, intervaloTela, func(passos int) {
		fmt.Printf("--- tela após %d instruções ---\n", passos)
		video.Desenhar(sim.Mapa)
	})
	if err != nil {
		fmt.Println("erro na execução:", err)
	}

	fmt.Println("--- tela final ---")
	video.Desenhar(sim.Mapa)
	fmt.Println("instruções executadas:", passos)
	fmt.Println("x8 =", sim.CPU.Regs[8], "(esperado 0, o contador do laço)")
	fmt.Printf("barramento: endereço=%#x dados=%d controle=%v\n",
		sim.Barramento.Endereco, sim.Barramento.Dados, sim.Barramento.Controle)
}