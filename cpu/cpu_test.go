package cpu

import (
	"errors"
	"testing"
)

// memoriaFalsa é um Barramento simples: guarda palavras num mapa e pode ser configurada para falhar em endereços específicos.
type memoriaFalsa struct {
	palavras map[uint32]uint32
	falhaEm  map[uint32]bool
}

// erroFalso é o erro que a memória falsa devolve nas falhas simuladas.
var erroFalso = errors.New("falha simulada")

func novaMemoriaFalsa() *memoriaFalsa {
	return &memoriaFalsa{
		palavras: map[uint32]uint32{},
		falhaEm:  map[uint32]bool{},
	}
}

func (m *memoriaFalsa) Read32(endereco uint32) (uint32, error) {
	if m.falhaEm[endereco] {
		return 0, erroFalso
	}
	return m.palavras[endereco], nil // endereço nunca escrito devolve 0
}

func (m *memoriaFalsa) Write32(endereco uint32, valor uint32) error {
	if m.falhaEm[endereco] {
		return erroFalso
	}
	m.palavras[endereco] = valor
	return nil
}

// novaCPU cria uma CPU com uma instrução guardada no endereço pc.
func novaCPU(pc, instrucao uint32) (*CPU, *memoriaFalsa) {
	mem := novaMemoriaFalsa()
	mem.palavras[pc] = instrucao
	c := New(mem)
	c.PC = pc
	return c, mem
}

// TestStepTipoR executa uma instrução Tipo-R por vez e confere x3 e o PC.
func TestStepTipoR(t *testing.T) {
	casos := []struct {
		nome      string
		instrucao uint32
		x1, x2    uint32 // valores iniciais dos registradores x1 e x2
		esperado  uint32 // valor esperado em x3
	}{
		{"add x3,x1,x2", 0x002081B3, 12, 10, 22},
		{"sub x3,x1,x2", 0x402081B3, 12, 10, 2},
		{"and x3,x1,x2", 0x0020F1B3, 12, 10, 8},
		{"or x3,x1,x2", 0x0020E1B3, 12, 10, 14},
		{"xor x3,x1,x2", 0x0020C1B3, 12, 10, 6},
		{"andn x3,x1,x2", 0x4020F1B3, 12, 10, 4}, // diferente do and (8)
		{"andn minúscula vira maiúscula", 0x4020F1B3, 'a', 0x20, 'A'},
		{"sll x3,x1,x2", 0x002091B3, 12, 10, 12288},
		{"srl entra zero", 0x0020D1B3, 0xF0000000, 4, 0x0F000000},
		{"sra copia o sinal", 0x4020D1B3, 0xF0000000, 4, 0xFF000000},
		{"slt -1 < 0", 0x0020A1B3, 0xFFFFFFFF, 0, 1},
		{"sltu 0xFFFFFFFF < 0", 0x0020B1B3, 0xFFFFFFFF, 0, 0},
	}

	for _, c := range casos {
		cpu, _ := novaCPU(0, c.instrucao)
		cpu.Regs[1], cpu.Regs[2] = c.x1, c.x2

		if err := cpu.Step(); err != nil {
			t.Errorf("%s: erro inesperado: %v", c.nome, err)
			continue
		}
		if cpu.Regs[3] != c.esperado {
			t.Errorf("%s: x3 = %#x, esperado %#x", c.nome, cpu.Regs[3], c.esperado)
		}
		if cpu.PC != 4 {
			t.Errorf("%s: PC = %d, esperado 4", c.nome, cpu.PC)
		}
	}
}

// TestStepTipoI executa instruções com imediato (opcode 0010011).
func TestStepTipoI(t *testing.T) {
	casos := []struct {
		nome      string
		instrucao uint32
		x1        uint32
		esperado  uint32 // valor esperado em x3
	}{
		{"addi x3,x1,-5", 0xFFB08193, 10, 5},
		{"andi x3,x1,15", 0x00F0F193, 0xAB, 0xB},
		{"ori x3,x1,15", 0x00F0E193, 0xF0, 0xFF},
		{"xori x3,x1,-1", 0xFFF0C193, 0, 0xFFFFFFFF},
		{"slli x3,x1,4", 0x00409193, 1, 16},
		{"srli x3,x1,4", 0x0040D193, 0xF0000000, 0x0F000000},
		{"srai x3,x1,4", 0x4040D193, 0xF0000000, 0xFF000000},
		{"slti x3,x1,0", 0x0000A193, 0xFFFFFFFF, 1},
		{"sltiu x3,x1,-1 (imm vira 0xFFFFFFFF)", 0xFFF0B193, 5, 1},
	}

	for _, c := range casos {
		cpu, _ := novaCPU(0, c.instrucao)
		cpu.Regs[1] = c.x1

		if err := cpu.Step(); err != nil {
			t.Errorf("%s: erro inesperado: %v", c.nome, err)
			continue
		}
		if cpu.Regs[3] != c.esperado {
			t.Errorf("%s: x3 = %#x, esperado %#x", c.nome, cpu.Regs[3], c.esperado)
		}
	}
}

// TestStepX0 confere que escrever no x0 é ignorado.
func TestStepX0(t *testing.T) {
	cpu, _ := novaCPU(0, 0x00500013) // addi x0, x0, 5
	if err := cpu.Step(); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if cpu.Regs[0] != 0 {
		t.Errorf("x0 = %d, deveria continuar 0", cpu.Regs[0])
	}
}

// TestStepSW confere que o sw grava rs2 no endereço rs1 + imediato.
func TestStepSW(t *testing.T) {
	// sw x2, 8(x1) com x1 = 100: grava em 108
	cpu, mem := novaCPU(0, 0x0020A423)
	cpu.Regs[1], cpu.Regs[2] = 100, 0xABCD
	if err := cpu.Step(); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if mem.palavras[108] != 0xABCD {
		t.Errorf("MEM[108] = %#x, esperado 0xABCD", mem.palavras[108])
	}

	// sw x2, -4(x1) com x1 = 100: deslocamento negativo, grava em 96
	cpu, mem = novaCPU(0, 0xFE20AE23)
	cpu.Regs[1], cpu.Regs[2] = 100, 0x1234
	if err := cpu.Step(); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if mem.palavras[96] != 0x1234 {
		t.Errorf("MEM[96] = %#x, esperado 0x1234", mem.palavras[96])
	}
}

// TestStepLW confere que o lw lê o endereço rs1 + imediato para rd.
func TestStepLW(t *testing.T) {
	// lw x3, 8(x1) com x1 = 100: lê de 108
	cpu, mem := novaCPU(0, 0x0080A183)
	cpu.Regs[1] = 100
	mem.palavras[108] = 0xCAFE
	if err := cpu.Step(); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if cpu.Regs[3] != 0xCAFE {
		t.Errorf("x3 = %#x, esperado 0xCAFE", cpu.Regs[3])
	}

	// lw x3, -4(x1) com x1 = 100: deslocamento negativo, lê de 96
	cpu, mem = novaCPU(0, 0xFFC0A183)
	cpu.Regs[1] = 100
	mem.palavras[96] = 0x5678
	if err := cpu.Step(); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if cpu.Regs[3] != 0x5678 {
		t.Errorf("x3 = %#x, esperado 0x5678", cpu.Regs[3])
	}
}

// TestStepBranches confere os seis desvios, tomados e não tomados.
// Todos têm deslocamento +16 e começam com PC = 0:
// se desvia, o PC vira 16; se não, vira 4.
func TestStepBranches(t *testing.T) {
	casos := []struct {
		nome      string
		instrucao uint32
		x1, x2    uint32
		desvia    bool
	}{
		{"beq iguais", 0x00208863, 5, 5, true},
		{"beq diferentes", 0x00208863, 5, 6, false},
		{"bne diferentes", 0x00209863, 5, 6, true},
		{"bne iguais", 0x00209863, 5, 5, false},
		// blt/bge comparam COM sinal: 0xFFFFFFFF vale -1
		{"blt -1 < 0", 0x0020C863, 0xFFFFFFFF, 0, true},
		{"blt 0 < -1", 0x0020C863, 0, 0xFFFFFFFF, false},
		{"bge 0 >= -1", 0x0020D863, 0, 0xFFFFFFFF, true},
		{"bge -1 >= 0", 0x0020D863, 0xFFFFFFFF, 0, false},
		// bltu/bgeu comparam SEM sinal: 0xFFFFFFFF é o maior número
		{"bltu 0 < 0xFFFFFFFF", 0x0020E863, 0, 0xFFFFFFFF, true},
		{"bltu 0xFFFFFFFF < 0", 0x0020E863, 0xFFFFFFFF, 0, false},
		{"bgeu 0xFFFFFFFF >= 0", 0x0020F863, 0xFFFFFFFF, 0, true},
		{"bgeu 0 >= 0xFFFFFFFF", 0x0020F863, 0, 0xFFFFFFFF, false},
	}

	for _, c := range casos {
		cpu, _ := novaCPU(0, c.instrucao)
		cpu.Regs[1], cpu.Regs[2] = c.x1, c.x2

		if err := cpu.Step(); err != nil {
			t.Errorf("%s: erro inesperado: %v", c.nome, err)
			continue
		}
		esperado := uint32(4)
		if c.desvia {
			esperado = 16
		}
		if cpu.PC != esperado {
			t.Errorf("%s: PC = %d, esperado %d", c.nome, cpu.PC, esperado)
		}
	}
}

// TestStepBranchParaTras confere um desvio com deslocamento negativo (um laço).
func TestStepBranchParaTras(t *testing.T) {
	cpu, _ := novaCPU(100, 0xFE000CE3) // beq x0, x0, -8 no endereço 100
	if err := cpu.Step(); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if cpu.PC != 92 {
		t.Errorf("PC = %d, esperado 92 (100 - 8)", cpu.PC)
	}
}

// TestStepInvalidas confere que instruções não suportadas dão erro
// e não alteram o PC nem os registradores.
func TestStepInvalidas(t *testing.T) {
	casos := []struct {
		nome      string
		instrucao uint32
	}{
		{"opcode desconhecido", 0x00000000},
		{"funct7 inválido no Tipo-R", 0x022081B3},
		{"lb (só lw é suportado)", 0x00008183},
		{"sh (só sw é suportado)", 0x00209023},
		{"branch com funct3 inexistente", 0x0020A863},
		{"ecall (só ebreak é suportado)", 0x00000073},
	}

	for _, c := range casos {
		cpu, _ := novaCPU(0, c.instrucao)
		cpu.Regs[3] = 99 // marcador: não pode mudar

		if err := cpu.Step(); err == nil {
			t.Errorf("%s: deveria dar erro", c.nome)
			continue
		}
		if cpu.PC != 0 {
			t.Errorf("%s: PC = %d, deveria continuar 0", c.nome, cpu.PC)
		}
		if cpu.Regs[3] != 99 {
			t.Errorf("%s: x3 mudou para %d", c.nome, cpu.Regs[3])
		}
	}
}

// TestStepEbreak confere que o ebreak marca Parou e não avança o PC.
func TestStepEbreak(t *testing.T) {
	cpu, _ := novaCPU(0, 0x00100073)
	if err := cpu.Step(); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !cpu.Parou {
		t.Error("Parou deveria ser true")
	}
	if cpu.PC != 0 {
		t.Errorf("PC = %d, deveria continuar apontando para o ebreak (0)", cpu.PC)
	}
}

// TestStepErroBarramento confere que falhas do barramento viram erro do Step
// e deixam o estado da CPU intacto.
func TestStepErroBarramento(t *testing.T) {
	// falha na busca da instrução
	cpu, mem := novaCPU(0, 0x002081B3)
	mem.falhaEm[0] = true
	err := cpu.Step()
	if !errors.Is(err, erroFalso) {
		t.Errorf("busca: erro = %v, esperado a falha simulada", err)
	}
	if cpu.PC != 0 {
		t.Errorf("busca: PC = %d, deveria continuar 0", cpu.PC)
	}

	// falha no lw: o registrador de destino não pode ser alterado
	cpu, mem = novaCPU(0, 0x0080A183) // lw x3, 8(x1)
	cpu.Regs[1], cpu.Regs[3] = 100, 99
	mem.falhaEm[108] = true
	err = cpu.Step()
	if !errors.Is(err, erroFalso) {
		t.Errorf("lw: erro = %v, esperado a falha simulada", err)
	}
	if cpu.Regs[3] != 99 {
		t.Errorf("lw: x3 mudou para %d", cpu.Regs[3])
	}
	if cpu.PC != 0 {
		t.Errorf("lw: PC = %d, deveria continuar 0", cpu.PC)
	}

	// falha no sw
	cpu, mem = novaCPU(0, 0x0020A423) // sw x2, 8(x1)
	cpu.Regs[1] = 100
	mem.falhaEm[108] = true
	err = cpu.Step()
	if !errors.Is(err, erroFalso) {
		t.Errorf("sw: erro = %v, esperado a falha simulada", err)
	}
	if cpu.PC != 0 {
		t.Errorf("sw: PC = %d, deveria continuar 0", cpu.PC)
	}
}

// ---------- interrupções ----------

// linhaFalsa é uma linha de interrupção que os testes ligam e desligam.
type linhaFalsa struct {
	pedindo bool
}

func (l *linhaFalsa) Interrupcao() bool {
	return l.pedindo
}

const (
	addiX1 = 0x00500093 // addi x1, x0, 5
	addiX2 = 0x00100113 // addi x2, x0, 1
	mret   = 0x30200073
)

// TestInterrupcaoDesviaParaARotina confere o que acontece ao interromper.
func TestInterrupcaoDesviaParaARotina(t *testing.T) {
	cpu, _ := novaCPU(0, addiX1)
	cpu.LinhaInterrupcao = &linhaFalsa{pedindo: true}

	if err := cpu.Step(); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if cpu.Regs[1] != 5 {
		t.Errorf("x1 = %d, a instrução deveria ter terminado antes de interromper", cpu.Regs[1])
	}
	if cpu.PC != VetorInterrupcao {
		t.Errorf("PC = %#x, esperado %#x (a rotina)", cpu.PC, VetorInterrupcao)
	}
	if cpu.PCSalvo != 4 {
		t.Errorf("PCSalvo = %d, esperado 4 (a próxima instrução)", cpu.PCSalvo)
	}
	if !cpu.EmTratamento {
		t.Error("EmTratamento deveria ser true")
	}
}

// TestSemInterrupcaoNaoDesvia confere os casos em que nada acontece.
func TestSemInterrupcaoNaoDesvia(t *testing.T) {
	// sem linha ligada (nil), como nos testes antigos
	cpu, _ := novaCPU(0, addiX1)
	if err := cpu.Step(); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if cpu.PC != 4 || cpu.EmTratamento {
		t.Errorf("sem linha: PC = %d, EmTratamento = %v", cpu.PC, cpu.EmTratamento)
	}

	// linha ligada, mas ninguém pedindo
	cpu, _ = novaCPU(0, addiX1)
	cpu.LinhaInterrupcao = &linhaFalsa{pedindo: false}
	if err := cpu.Step(); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if cpu.PC != 4 || cpu.EmTratamento {
		t.Errorf("linha quieta: PC = %d, EmTratamento = %v", cpu.PC, cpu.EmTratamento)
	}
}

// TestInterrupcaoNaoAninha confere que, dentro da rotina, novas interrupções esperam.
func TestInterrupcaoNaoAninha(t *testing.T) {
	cpu, mem := novaCPU(0, addiX1)
	mem.palavras[VetorInterrupcao] = addiX2
	cpu.LinhaInterrupcao = &linhaFalsa{pedindo: true}

	cpu.Step()                         // entra na rotina
	if err := cpu.Step(); err != nil { // executa uma instrução da rotina
		t.Fatalf("erro inesperado: %v", err)
	}

	if cpu.Regs[2] != 1 {
		t.Errorf("x2 = %d, a rotina deveria ter executado", cpu.Regs[2])
	}
	if cpu.PC != VetorInterrupcao+4 {
		t.Errorf("PC = %#x, esperado %#x (segue na rotina)", cpu.PC, VetorInterrupcao+4)
	}
	if cpu.PCSalvo != 4 {
		t.Errorf("PCSalvo = %d, não deveria ser sobrescrito (esperado 4)", cpu.PCSalvo)
	}
}

// TestMretVoltaAoPrograma confere o retorno ao ponto certo.
func TestMretVoltaAoPrograma(t *testing.T) {
	cpu, mem := novaCPU(0, addiX1)
	mem.palavras[VetorInterrupcao] = mret
	linha := &linhaFalsa{pedindo: true}
	cpu.LinhaInterrupcao = linha

	cpu.Step()                         // entra na rotina
	linha.pedindo = false              // a rotina atendeu o pedido
	if err := cpu.Step(); err != nil { // executa o MRET
		t.Fatalf("erro inesperado: %v", err)
	}

	if cpu.PC != 4 {
		t.Errorf("PC = %d, esperado 4 (onde o programa parou)", cpu.PC)
	}
	if cpu.EmTratamento {
		t.Error("EmTratamento deveria voltar a false")
	}
}

// TestInterrupcaoPendenteDepoisDoMret confere que uma interrupção que continua pedindo é tratada logo depois do retorno.
func TestInterrupcaoPendenteDepoisDoMret(t *testing.T) {
	cpu, mem := novaCPU(0, addiX1)
	mem.palavras[VetorInterrupcao] = mret
	cpu.LinhaInterrupcao = &linhaFalsa{pedindo: true} // nunca para de pedir

	cpu.Step() // entra na rotina
	cpu.Step() // MRET, e logo interrompe de novo

	if cpu.PC != VetorInterrupcao || !cpu.EmTratamento {
		t.Errorf("PC = %#x, EmTratamento = %v; esperado uma nova entrada na rotina",
			cpu.PC, cpu.EmTratamento)
	}
	if cpu.PCSalvo != 4 {
		t.Errorf("PCSalvo = %d, esperado 4 (o programa ainda não avançou)", cpu.PCSalvo)
	}
}

// TestMretForaDeRotina confere que o MRET sozinho é uma instrução inválida.
func TestMretForaDeRotina(t *testing.T) {
	cpu, _ := novaCPU(0, mret)

	if err := cpu.Step(); err == nil {
		t.Error("MRET fora de uma rotina deveria dar erro")
	}
	if cpu.PC != 0 {
		t.Errorf("PC = %d, deveria continuar 0", cpu.PC)
	}
}

// TestInterrupcaoDepoisDeDesvio confere que o programa volta ao destino do desvio.
func TestInterrupcaoDepoisDeDesvio(t *testing.T) {
	cpu, _ := novaCPU(0, 0x00000863) // beq x0, x0, 16: sempre desvia para 16
	cpu.LinhaInterrupcao = &linhaFalsa{pedindo: true}

	if err := cpu.Step(); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if cpu.PCSalvo != 16 {
		t.Errorf("PCSalvo = %d, esperado 16 (o destino do desvio, e não 4)", cpu.PCSalvo)
	}
}

// TestErroNaoInterrompe confere que uma instrução que falhou não dispara a rotina.
func TestErroNaoInterrompe(t *testing.T) {
	cpu, _ := novaCPU(0, 0x00000000) // opcode inválido
	cpu.LinhaInterrupcao = &linhaFalsa{pedindo: true}

	if err := cpu.Step(); err == nil {
		t.Fatal("deveria dar erro")
	}
	if cpu.EmTratamento || cpu.PC != 0 {
		t.Errorf("EmTratamento = %v, PC = %d; a CPU não deveria ter mudado", cpu.EmTratamento, cpu.PC)
	}
}

// TestEbreakNaoInterrompe confere que, ao parar, a CPU não entra na rotina.
func TestEbreakNaoInterrompe(t *testing.T) {
	cpu, _ := novaCPU(0, 0x00100073) // ebreak
	cpu.LinhaInterrupcao = &linhaFalsa{pedindo: true}

	if err := cpu.Step(); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !cpu.Parou || cpu.EmTratamento || cpu.PC != 0 {
		t.Errorf("Parou = %v, EmTratamento = %v, PC = %d", cpu.Parou, cpu.EmTratamento, cpu.PC)
	}
}

// TestEntrarESairNaoMexeNosRegistradores confere que a CPU não salva nem apaga registradores: isso é trabalho da rotina.
func TestEntrarESairNaoMexeNosRegistradores(t *testing.T) {
	cpu, mem := novaCPU(0, addiX1)
	mem.palavras[VetorInterrupcao] = mret
	linha := &linhaFalsa{pedindo: true}
	cpu.LinhaInterrupcao = linha
	cpu.Regs[5] = 77

	cpu.Step() // entra
	linha.pedindo = false
	cpu.Step() // MRET

	if cpu.Regs[5] != 77 {
		t.Errorf("x5 = %d, esperado 77", cpu.Regs[5])
	}
	if cpu.Regs[1] != 5 {
		t.Errorf("x1 = %d, esperado 5", cpu.Regs[1])
	}
}
