package mem

import (
	"strings"
	"testing"
)

// TestMapaPadraoRoteamento confere que cada região guarda o seu dado, sem colidir com as outras.
func TestMapaPadraoRoteamento(t *testing.T) {
	m, err := NovoMapaPadrao()
	if err != nil {
		t.Fatalf("erro ao montar o mapa: %v", err)
	}

	// primeiro e último endereço de cada região com dono
	casos := []struct {
		nome     string
		endereco uint32
		valor    uint32
	}{
		{"RAM, primeiro", 0x00000, 11},
		{"RAM, último", 0x7FFFC, 22},
		{"VRAM, primeiro", 0x80000, 33},
		{"VRAM, último", 0x8FFFC, 44},
		{"Periféricos, primeiro", 0x9FC00, 55},
		{"Periféricos, último", 0x9FFFC, 66},
	}

	for _, c := range casos {
		if err := m.Write32(c.endereco, c.valor); err != nil {
			t.Errorf("%s: Write32(%#x): erro inesperado: %v", c.nome, c.endereco, err)
		}
	}

	// só depois de gravar tudo é que confere se duas regiões dividissem o mesmo lugar, um valor teria sobrescrito o outro
	for _, c := range casos {
		valor, err := m.Read32(c.endereco)
		if err != nil {
			t.Errorf("%s: Read32(%#x): erro inesperado: %v", c.nome, c.endereco, err)
		} else if valor != c.valor {
			t.Errorf("%s: Read32(%#x) = %d, esperado %d", c.nome, c.endereco, valor, c.valor)
		}
	}
}

// TestMapaPadraoLimites confere os endereços que não podem ser acessados.
func TestMapaPadraoLimites(t *testing.T) {
	m, err := NovoMapaPadrao()
	if err != nil {
		t.Fatalf("erro ao montar o mapa: %v", err)
	}

	casos := []struct {
		nome     string
		endereco uint32
		trecho   string // pedaço que a mensagem de erro deve conter
	}{
		{"reservada, primeiro", 0x90000, "reservada"},
		{"reservada, último", 0x9FBFC, "reservada"},
		{"depois dos periféricos", 0xA0000, "não está mapeado"},
		{"fim do espaço de endereços", 0xFFFFFFFC, "não está mapeado"},
	}

	for _, c := range casos {
		_, errLeitura := m.Read32(c.endereco)
		errEscrita := m.Write32(c.endereco, 1)

		if errLeitura == nil || !strings.Contains(errLeitura.Error(), c.trecho) {
			t.Errorf("%s: Read32(%#x): erro = %v, esperado conter %q", c.nome, c.endereco, errLeitura, c.trecho)
		}
		if errEscrita == nil || !strings.Contains(errEscrita.Error(), c.trecho) {
			t.Errorf("%s: Write32(%#x): erro = %v, esperado conter %q", c.nome, c.endereco, errEscrita, c.trecho)
		}
	}
}

// TestMapaDonoUnico confere se cada acesso chega só ao dispositivo dono do endereço.
func TestMapaDonoUnico(t *testing.T) {
	ramA := NovaRAM(0x000, 0x100) // dona de 0x000 até 0x0FF
	ramB := NovaRAM(0x100, 0x100) // dona de 0x100 até 0x1FF

	m := &Mapa{}
	if err := m.Adicionar("A", 0x000, 0x100, ramA); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if err := m.Adicionar("B", 0x100, 0x100, ramB); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	// o acesso em 0x104 pertence à B
	if err := m.Write32(0x104, 77); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if valor, err := ramB.Read32(0x104); err != nil || valor != 77 {
		t.Errorf("B deveria ter recebido o valor: %d, %v", valor, err)
	}
	// a A não pode ter visto esse endereço (nem sequer o atende)
	if _, err := ramA.Read32(0x104); err == nil {
		t.Error("A não deveria atender o endereço 0x104")
	}
}

// TestMapaSobreposicao confere se regiões que se cruzam são recusadas.
func TestMapaSobreposicao(t *testing.T) {
	m := &Mapa{}
	if err := m.Adicionar("A", 0x100, 0x100, NovaRAM(0x100, 0x100)); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	// A ocupa 0x100 até 0x1FF. Todas estas se cruzam com ela.
	cruzam := []struct {
		nome          string
		base, tamanho uint32
	}{
		{"idêntica", 0x100, 0x100},
		{"começa antes e invade", 0x80, 0x100},
		{"começa dentro e passa", 0x180, 0x100},
		{"contida", 0x120, 0x10},
		{"contém a outra", 0x0, 0x400},
	}
	for _, c := range cruzam {
		if err := m.Adicionar(c.nome, c.base, c.tamanho, nil); err == nil {
			t.Errorf("%s: deveria ser recusada", c.nome)
		}
	}
	// as recusadas não podem ter entrado no mapa
	if len(m.regioes) != 1 {
		t.Errorf("o mapa tem %d regiões, esperado 1", len(m.regioes))
	}

	// regiões coladas, sem cruzar, são aceitas
	if err := m.Adicionar("logo depois", 0x200, 0x100, nil); err != nil {
		t.Errorf("região logo depois deveria ser aceita: %v", err)
	}
	if err := m.Adicionar("logo antes", 0x000, 0x100, nil); err != nil {
		t.Errorf("região logo antes deveria ser aceita: %v", err)
	}
}

// TestMapaCarregar confere o carregamento de programa pelo mapa.
func TestMapaCarregar(t *testing.T) {
	m, err := NovoMapaPadrao()
	if err != nil {
		t.Fatalf("erro ao montar o mapa: %v", err)
	}

	if err := m.Load(0, []uint32{10, 20, 30}); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	for i, esperado := range []uint32{10, 20, 30} {
		if valor, _ := m.Read32(uint32(i) * 4); valor != esperado {
			t.Errorf("posição %d = %d, esperado %d", i, valor, esperado)
		}
	}

	// carregar na região reservada tem que falhar
	if err := m.Load(0x90000, []uint32{1}); err == nil {
		t.Error("carregar na região reservada deveria dar erro")
	}
}

// TestMapaComPerifericos confere que o dispositivo dado ocupa a região de periféricos, e só ela.
func TestMapaComPerifericos(t *testing.T) {
	perifericos := NovaRAM(0x9FC00, 0x400)
	m, err := NovoMapaCom(perifericos)
	if err != nil {
		t.Fatalf("erro ao montar o mapa: %v", err)
	}

	// a escrita em 0x9FC04 chega ao dispositivo
	if err := m.Write32(0x9FC04, 55); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if valor, err := perifericos.Read32(0x9FC04); err != nil || valor != 55 {
		t.Errorf("o dispositivo recebeu %d, %v; esperado 55", valor, err)
	}

	// as outras regiões continuam donas dos seus endereços
	if err := m.Write32(0x100, 7); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if _, err := perifericos.Read32(0x100); err == nil {
		t.Error("o dispositivo dos periféricos não deveria atender o endereço 0x100")
	}
	if _, err := m.Read32(0x90000); err == nil {
		t.Error("a região reservada deveria continuar recusando acesso")
	}
}
