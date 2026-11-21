package mem

import "testing"

// TestRAMPrimeiroEUltimoEndereco confere as duas pontas da memória.
func TestRAMPrimeiroEUltimoEndereco(t *testing.T) {
	r := NovaRAM(0x100, 16) // 4 palavras: 0x100, 0x104, 0x108, 0x10C

	for _, endereco := range []uint32{0x100, 0x10C} {
		if err := r.Write32(endereco, 0xDEADBEEF); err != nil {
			t.Errorf("Write32(%#x): erro inesperado: %v", endereco, err)
			continue
		}
		valor, err := r.Read32(endereco)
		if err != nil {
			t.Errorf("Read32(%#x): erro inesperado: %v", endereco, err)
		} else if valor != 0xDEADBEEF {
			t.Errorf("Read32(%#x) = %#x, esperado 0xDEADBEEF", endereco, valor)
		}
	}
}

// TestRAMForaDoLimite confere que endereços fora da memória dão erro.
func TestRAMForaDoLimite(t *testing.T) {
	r := NovaRAM(0x100, 16) // válidos: 0x100 até 0x10C

	// antes da base, logo depois do fim, o começo do espaço e o fim dele
	for _, endereco := range []uint32{0xFC, 0x110, 0x0, 0xFFFFFFFC} {
		if _, err := r.Read32(endereco); err == nil {
			t.Errorf("Read32(%#x): deveria dar erro", endereco)
		}
		if err := r.Write32(endereco, 1); err == nil {
			t.Errorf("Write32(%#x): deveria dar erro", endereco)
		}
	}
}

// TestRAMDesalinhado confere que endereço que não é múltiplo de 4 dá erro.
func TestRAMDesalinhado(t *testing.T) {
	r := NovaRAM(0x100, 16)

	for _, endereco := range []uint32{0x101, 0x102, 0x103, 0x10E} {
		if _, err := r.Read32(endereco); err == nil {
			t.Errorf("Read32(%#x): deveria dar erro de alinhamento", endereco)
		}
		if err := r.Write32(endereco, 1); err == nil {
			t.Errorf("Write32(%#x): deveria dar erro de alinhamento", endereco)
		}
	}
}

// TestRAMPreservaDados confere que escrever numa posição não mexe nas vizinhas.
func TestRAMPreservaDados(t *testing.T) {
	r := NovaRAM(0x100, 16)

	// a memória nova começa zerada
	for _, endereco := range []uint32{0x100, 0x104, 0x108, 0x10C} {
		if valor, _ := r.Read32(endereco); valor != 0 {
			t.Errorf("Read32(%#x) = %#x, a memória nova deveria ser 0", endereco, valor)
		}
	}

	r.Write32(0x104, 7)
	r.Write32(0x104, 9) // sobrescreve

	esperado := map[uint32]uint32{0x100: 0, 0x104: 9, 0x108: 0, 0x10C: 0}
	for endereco, valor := range esperado {
		if obtido, _ := r.Read32(endereco); obtido != valor {
			t.Errorf("Read32(%#x) = %d, esperado %d", endereco, obtido, valor)
		}
	}
}

// TestRAMConversaoDeEndereco confere a conta endereço -> posição na lista.
func TestRAMConversaoDeEndereco(t *testing.T) {
	r := NovaRAM(0x80000, 0x10000) // como a VRAM do mapa

	if err := r.Write32(0x80004, 42); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	// (0x80004 - 0x80000) / 4 = posição 1
	if r.palavras[1] != 42 {
		t.Errorf("palavras[1] = %d, esperado 42", r.palavras[1])
	}
	if _, err := r.Read32(0x0); err == nil {
		t.Error("0x0 está antes da base: deveria dar erro")
	}
}

// TestRAMCarregar confere o carregamento de um programa.
func TestRAMCarregar(t *testing.T) {
	r := NovaRAM(0, 16)

	if err := r.Load(0, []uint32{1, 2, 3}); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	esperado := []uint32{1, 2, 3, 0}
	for i, valor := range esperado {
		if obtido, _ := r.Read32(uint32(i) * 4); obtido != valor {
			t.Errorf("posição %d = %d, esperado %d", i, obtido, valor)
		}
	}

	// programa que não cabe a partir do endereço 4 (4 palavras em 3 posições)
	if err := r.Load(4, []uint32{1, 2, 3, 4}); err == nil {
		t.Error("programa grande demais: deveria dar erro")
	}
	// endereço inicial desalinhado
	if err := r.Load(2, []uint32{1}); err == nil {
		t.Error("endereço inicial desalinhado: deveria dar erro")
	}
}