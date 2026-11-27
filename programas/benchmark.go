package programas

// SomaVetor é um programa para medir desempenho. Preenche um vetor de 256 palavras em 0x2000 com 1..256 e depois soma o vetor inteiro 100 vezes.

// No fim, x9 = 100 * (1 + 2 + ... + 256) = 3289600. Executa 129432 instruções, e o vetor ocupa todas as 256 linhas da cache.
var SomaVetor = []uint32{
	0x00100293, // 0x00 addi x5, x0, 1
	0x00D29293, // 0x04 slli x5, x5, 13        x5 = 0x2000 (base do vetor)
	0x10000313, // 0x08 addi x6, x0, 256       N = 256 palavras
	0x00000393, // 0x0C addi x7, x0, 0         i = 0
	0x00028413, // 0x10 addi x8, x5, 0         ptr = base
	0x00138393, // 0x14 fill: addi x7, x7, 1
	0x00742023, // 0x18 sw   x7, 0(x8)         vetor[i] = i + 1
	0x00440413, // 0x1C addi x8, x8, 4
	0xFE639AE3, // 0x20 bne  x7, x6, fill
	0x00000493, // 0x24 addi x9, x0, 0         soma = 0
	0x06400513, // 0x28 addi x10, x0, 100      repetições
	0x00028413, // 0x2C rep: addi x8, x5, 0    ptr = base
	0x00000393, // 0x30 addi x7, x0, 0         i = 0
	0x00042583, // 0x34 soma: lw x11, 0(x8)
	0x00B484B3, // 0x38 add  x9, x9, x11       soma += vetor[i]
	0x00440413, // 0x3C addi x8, x8, 4
	0x00138393, // 0x40 addi x7, x7, 1
	0xFE6398E3, // 0x44 bne  x7, x6, soma
	0xFFF50513, // 0x48 addi x10, x10, -1
	0xFE0510E3, // 0x4C bne  x10, x0, rep
	0x00100073, // 0x50 ebreak
}
