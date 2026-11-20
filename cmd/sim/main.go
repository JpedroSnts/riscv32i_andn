package main

import (
	"fmt"
	"riscv/barramento"
	"riscv/cpu"
	"riscv/mem"
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
	programa := []uint32{
		0x02000513, // addi x10, x0, 32     máscara do bit 5
		0x06100393, // addi x7, x0, 97      x7 = 'a'
		0x10000213, // addi x4, x0, 256     endereço na RAM
		0x00722023, // sw   x7, 0(x4)       guarda 'a' na RAM
		0x00022383, // lw   x7, 0(x4)       lê de volta
		0x00100313, // addi x6, x0, 1
		0x01331313, // slli x6, x6, 19      x6 = 0x80000 (VRAM)
		0x00800413, // addi x8, x0, 8       contador: 8 letras
		0x40A3F4B3, // andn x9, x7, x10     início do laço: maiúscula
		0x00932023, // sw   x9, 0(x6)       escreve na VRAM
		0x00138393, // addi x7, x7, 1       próxima letra
		0x00430313, // addi x6, x6, 4       próxima posição da VRAM
		0xFFF40413, // addi x8, x8, -1      contador - 1
		0xFE0416E3, // bne  x8, x0, -20     se x8 != 0, volta ao andn
		0x00100073, // ebreak               fim do programa
	}
	if err := mapa.Carregar(0, programa); err != nil {
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
			video.Mostrar(mapa)
		}
	}

	if !processador.Parou {
		fmt.Println("limite de passos atingido: o programa não terminou")
	}

	fmt.Println("--- tela final ---")
	video.Mostrar(mapa)
	fmt.Println("instruções executadas:", passos)
	fmt.Println("x8 =", processador.Regs[8], "(esperado 0, o contador do laço)")
	fmt.Printf("barramento: endereço=%#x dados=%d controle=%v\n",
		sistema.Endereco, sistema.Dados, sistema.Controle)
}