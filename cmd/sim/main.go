package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"riscv/cpu"
	"riscv/programas"
	"riscv/simulador"
	"riscv/video"
)

// enderecoContador é onde a rotina de tratamento conta as interrupções.
const enderecoContador = 0x740

func main() {
	interrupcao := flag.Bool("interrupcao", false,
		"roda a demonstração da interrupção por teclado (digite i e Enter)")
	pausa := flag.Duration("pausa", 30*time.Millisecond,
		"pausa entre as instruções no modo -interrupcao")
	flag.Parse()

	if *interrupcao {
		demonstrarInterrupcao(*pausa)
		return
	}
	demonstrarAlfabeto()
}

// demonstrarAlfabeto roda o programa que escreve ABCDEFGH na VRAM com um laço.
func demonstrarAlfabeto() {
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
	fmt.Printf("cache: %d hits, %d misses\n", sim.Cache.Hits, sim.Cache.Misses)
}

// demonstrarInterrupcao roda o programa principal com a rotina de tratamento carregada e lê o teclado do terminal: digite i e Enter para interromper.
func demonstrarInterrupcao(pausa time.Duration) {
	sim, err := simulador.Novo(programas.InterrupcaoPrincipal)
	if err != nil {
		fmt.Println("erro ao montar o simulador:", err)
		return
	}
	// a rotina de tratamento fica no endereço que a CPU usa nas interrupções
	if err := sim.Mapa.Load(cpu.VetorInterrupcao, programas.InterrupcaoRotina); err != nil {
		fmt.Println("erro ao carregar a rotina de tratamento:", err)
		return
	}

	// lê o terminal em paralelo com a CPU
	go sim.Teclado.LerDe(os.Stdin)

	fmt.Println("Demonstração da interrupção por teclado.")
	fmt.Println("Digite i e Enter a qualquer momento para interromper o programa.")
	fmt.Println()

	estavaNaRotina := false
	passos, err := sim.Executar(10000, 1, func(passos int) {
		switch {
		case sim.CPU.PC == cpu.VetorInterrupcao:
			fmt.Printf("[instrução %d] interrupção: a CPU foi para a rotina em %#x\n",
				passos, sim.CPU.PC)
		case estavaNaRotina && !sim.CPU.EmTratamento:
			fmt.Printf("[instrução %d] mret: o programa continua em %#x\n",
				passos, sim.CPU.PC)
			video.Desenhar(sim.Mapa)
		case passos%50 == 0:
			fmt.Printf("[instrução %d] laço principal: x8 = %d\n",
				passos, sim.CPU.Regs[8])
		}
		estavaNaRotina = sim.CPU.EmTratamento

		time.Sleep(pausa) // dá tempo de digitar
	})
	if err != nil {
		fmt.Println("erro na execução:", err)
	}

	atendidas, _ := sim.Mapa.Read32(enderecoContador)

	fmt.Println()
	fmt.Println("--- tela final ---")
	video.Desenhar(sim.Mapa)
	fmt.Println("instruções executadas:", passos)
	fmt.Println("x8 =", sim.CPU.Regs[8], "(esperado 200, o laço terminou)")
	fmt.Println("interrupções atendidas:", atendidas)
	fmt.Printf("cache: %d hits, %d misses\n", sim.Cache.Hits, sim.Cache.Misses)
}
