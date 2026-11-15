package main

import (
	"fmt"
	"riscv/cpu"
	"riscv/mem"
)

func main() {
	// RAM principal: começa em 0x00000 e tem 512 KB (0x80000 bytes).
	ram := mem.NovaRAM(0x00000, 0x80000)

	// Programa DEMO (palavras de 32 bits).
	programa := []uint32{
		0x00C00093, // addi x1, x0, 12
		0x00A00113, // addi x2, x0, 10
		0x4020F1B3, // andn x3, x1, x2
		0x10000213, // addi x4, x0, 256
		0x00322023, // sw   x3, 0(x4)
		0x00022283, // lw   x5, 0(x4)
	}

	if err := ram.Load(0, programa); err != nil {
		fmt.Println("erro ao carregar o programa:", err)
		return
	}

	// A CPU usa a RAM como barramento.
	processador := cpu.New(ram)

	// Executa uma instrução por vez, num limite fixo de passos.
	// Aqui o limite é o tamanho do programa, que só funciona porque ele não tem desvios. Com branches, será preciso outra condição de parada.
	for n := 0; n < len(programa); n++ {
		if err := processador.Step(); err != nil {
			fmt.Println("erro na CPU:", err)
			return
		}
	}
	
	// Confere os resultados.
	guardado, _ := ram.Read32(256)
	fmt.Println("x1 =", processador.Regs[1], "(esperado 12)")
	fmt.Println("x2 =", processador.Regs[2], "(esperado 10)")
	fmt.Println("x3 =", processador.Regs[3], "(esperado 4, resultado do andn)")
	fmt.Println("MEM[256] =", guardado, "(esperado 4, gravado pelo sw)")
	fmt.Println("x5 =", processador.Regs[5], "(esperado 4, lido pelo lw)")
}