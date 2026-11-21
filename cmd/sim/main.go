package main

import (
	"fmt"
	"riscv/barramento"
	"riscv/cpu"
	"riscv/mem"
	"riscv/programas"
	"riscv/video"
)

func main() {
	// Mapa de memória do enunciado: RAM, VRAM, reservada e periféricos.
	mapa, err := mem.NovoMapaPadrao()
	if err != nil {
		fmt.Println("erro ao montar o mapa:", err)
		return
	}

	// Programa de teste: escreve "ABCDEFGH" na VRAM com um laço.
	// Converte cada letra de minúscula para maiúscula com ANDN (apaga o bit 5).
	programa := programas.Alfabeto
	
	if err := mapa.Load(0, programa); err != nil {
		fmt.Println("erro ao carregar o programa:", err)
		return
	}

	// O barramento fica entre a CPU e o mapa de memória.
	sistema := barramento.NovoSistema(mapa)
	processador := cpu.New(sistema)

	const intervaloTela = 20   // a VRAM é mostrada a cada 20 instruções
	const limitePassos = 10000 // trava de segurança contra laço infinito

	passos := 0
	for !processador.Parou && passos < limitePassos {
		if err := processador.Step(); err != nil {
			fmt.Println("erro na CPU:", err)
			return
		}
		passos++

		if passos%intervaloTela == 0 {
			fmt.Printf("--- tela após %d instruções ---\n", passos)
			video.Desenhar(mapa)
		}
	}

	if !processador.Parou {
		fmt.Println("limite de passos atingido: o programa não terminou")
	}

	fmt.Println("--- tela final ---")
	video.Desenhar(mapa)
	fmt.Println("instruções executadas:", passos)
	fmt.Println("x8 =", processador.Regs[8], "(esperado 0, o contador do laço)")
	fmt.Printf("barramento: endereço=%#x dados=%d controle=%v\n",
		sistema.Endereco, sistema.Dados, sistema.Controle)
}