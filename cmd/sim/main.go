package main

import (
	"fmt"
	"riscv/cpu"
	"riscv/mem"
	"riscv/barramento"
	"riscv/video"
)

func main() {
	// Mapa de memória do enunciado: RAM, VRAM, reservada e periféricos.
	mapa, err := mem.NovoMapaPadrao()
	if err != nil {
		fmt.Println("erro ao montar o mapa:", err)
		return
	}

	programa := []uint32{
		0x00C00093, // addi x1, x0, 12
		0x00A00113, // addi x2, x0, 10
		0x4020F1B3, // andn x3, x1, x2
		0x10000213, // addi x4, x0, 256
		0x00322023, // sw   x3, 0(x4)
		0x00022283, // lw   x5, 0(x4)
		0x00100313, // addi x6, x0, 1
		0x01331313, // slli x6, x6, 19     (x6 = 0x80000, início da VRAM)
		0x04800393, // addi x7, x0, 72     (72 = 'H' em ASCII)
		0x00732023, // sw   x7, 0(x6)      (escreve na VRAM)
		0x04900393, // addi x7, x0, 73     (73 = 'I' em ASCII)
		0x00732223, // sw   x7, 4(x6)      (próxima posição da VRAM)
	}
	if err := mapa.Carregar(0, programa); err != nil {
		fmt.Println("erro ao carregar o programa:", err)
		return
	}

	// O barramento fica entre a CPU e o mapa de memória.
	sistema := barramento.NovoSistema(mapa)
	processador := cpu.New(sistema)

	// A cada intervaloTela instruções, a VRAM é mostrada no terminal.
	const intervaloTela = 4

	for n := 1; n <= len(programa); n++ {
		if err := processador.Step(); err != nil {
			fmt.Println("erro na CPU:", err)
			return
		}
		if n%intervaloTela == 0 {
			fmt.Printf("--- tela após %d instruções ---\n", n)
			video.Mostrar(mapa)
		}
	}

	fmt.Println("--- tela final ---")
	video.Mostrar(mapa)
	// Estado das linhas depois do último acesso (o sw na VRAM).
	fmt.Printf("barramento: endereço=%#x dados=%d controle=%v\n",sistema.Endereco, sistema.Dados, sistema.Controle)

	fmt.Println("x3 =", processador.Regs[3], "(esperado 4, resultado do andn)")
	fmt.Println("x5 =", processador.Regs[5], "(esperado 4, lido da RAM)")

	vram, _ := mapa.Read32(0x80000)
	fmt.Println("VRAM[0x80000] =", vram, "(esperado 72, gravado pelo sw)")

	// Testes das regiões especiais.
	if _, err := mapa.Read32(0x90000); err != nil {
		fmt.Println("região reservada:", err)
	}
	if _, err := mapa.Read32(0x9FC00); err == nil {
		fmt.Println("periféricos: acesso permitido")
	}
	if _, err := mapa.Read32(0xA0000); err != nil {
		fmt.Println("fora do mapa:", err)
	}
}