package programas

// Alfabeto escreve "ABCDEFGH" na VRAM com um laço.
// Converte cada letra de minúscula para maiúscula com ANDN (apaga o bit 5).
// Usa ULA, ANDN, LW, SW, desvio, VRAM e termina com EBREAK.
var Alfabeto = []uint32{
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
