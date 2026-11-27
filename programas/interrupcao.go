package programas

// InterrupcaoPrincipal é o programa principal da demonstração de interrupção.

// Fica em 0x0000. Liga as interrupções do teclado, conta de 1 a 200 e termina com um 'F' na linha 1 da tela. Os registradores x28, x29 e x30 recebem valores conhecidos porque a rotina de tratamento também os usa.

var InterrupcaoPrincipal = []uint32{
	0x00A00513, // 0x00 addi x10, x0, 10
	0x01051513, // 0x04 slli x10, x10, 16
	0xC0050513, // 0x08 addi x10, x10, -1024   x10 = 0x9FC00 (teclado)
	0x00100593, // 0x0C addi x11, x0, 1
	0x00B52423, // 0x10 sw   x11, 8(x10)       CONTROLE = 1: liga as interrupções
	0x00100313, // 0x14 addi x6, x0, 1
	0x01331313, // 0x18 slli x6, x6, 19        x6 = 0x80000 (VRAM)
	0x04D00E13, // 0x1C addi x28, x0, 77       marcadores que a rotina precisa preservar
	0x05800E93, // 0x20 addi x29, x0, 88
	0x06300F13, // 0x24 addi x30, x0, 99
	0x0C800493, // 0x28 addi x9, x0, 200       limite do laço
	0x00140413, // 0x2C addi x8, x8, 1         início do laço: contador + 1
	0xFE941EE3, // 0x30 bne  x8, x9, -4        volta enquanto x8 != 200
	0x04600393, // 0x34 addi x7, x0, 70        'F'
	0x0A732023, // 0x38 sw   x7, 160(x6)       escreve na linha 1 da tela
	0x00100073, // 0x3C ebreak                 fim do programa
}

// InterrupcaoRotina é a rotina de tratamento. Fica em 0x1000 (VetorInterrupcao).

// Salva x28, x29 e x30 em 0x0700, lê a tecla (o que limpa a pendência), conta

// a interrupção em 0x0740, escreve um '*' na VRAM, restaura os registradores e volta ao programa com MRET.
var InterrupcaoRotina = []uint32{
	0x71C02023, // 0x1000 sw   x28, 0x700(x0)   salva x28
	0x71D02223, // 0x1004 sw   x29, 0x704(x0)   salva x29
	0x71E02423, // 0x1008 sw   x30, 0x708(x0)   salva x30
	0x00A00E13, // 0x100C addi x28, x0, 10
	0x010E1E13, // 0x1010 slli x28, x28, 16
	0xC00E0E13, // 0x1014 addi x28, x28, -1024  x28 = 0x9FC00 (teclado)
	0x004E2E83, // 0x1018 lw   x29, 4(x28)      lê a TECLA: limpa a pendência
	0x74002E83, // 0x101C lw   x29, 0x740(x0)   x29 = contador de interrupções
	0x001E8E93, // 0x1020 addi x29, x29, 1      contador + 1
	0x75D02023, // 0x1024 sw   x29, 0x740(x0)   guarda o contador
	0x00100F13, // 0x1028 addi x30, x0, 1
	0x013F1F13, // 0x102C slli x30, x30, 19     x30 = 0x80000 (VRAM)
	0x002E9E13, // 0x1030 slli x28, x29, 2      x28 = contador * 4
	0x01CF0F33, // 0x1034 add  x30, x30, x28    x30 = VRAM + contador * 4
	0x02A00E13, // 0x1038 addi x28, x0, 42      '*'
	0xFFCF2E23, // 0x103C sw   x28, -4(x30)     escreve na posição contador - 1
	0x70002E03, // 0x1040 lw   x28, 0x700(x0)   restaura x28
	0x70402E83, // 0x1044 lw   x29, 0x704(x0)   restaura x29
	0x70802F03, // 0x1048 lw   x30, 0x708(x0)   restaura x30
	0x30200073, // 0x104C mret                  volta ao programa
}
