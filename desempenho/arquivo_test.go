package desempenho

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"riscv/programas"
)

// iguais compara duas listas de instruções.
func iguais(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestLerPalavrasFormatos confere todos os jeitos aceitos de escrever o programa.
func TestLerPalavrasFormatos(t *testing.T) {
	casos := []struct {
		nome     string
		texto    string
		esperado []uint32
	}{
		{"sem prefixo", "00100293\n00100073\n", []uint32{0x00100293, 0x00100073}},
		{"com prefixo", "0x00100293\n0X00100073\n", []uint32{0x00100293, 0x00100073}},
		{"maiúsculas e minúsculas", "0xfe639ae3\n0xFE639AE3\n", []uint32{0xFE639AE3, 0xFE639AE3}},
		{"menos de 8 dígitos", "0x1\nFF\n", []uint32{1, 0xFF}},
		{"em branco e espaços", "\n  0x00100293  \n\n\t0x00100073\n", []uint32{0x00100293, 0x00100073}},
		{"comentários", "// programa\n0x00100293 // addi\n# outro\n0x00100073 # ebreak\n", []uint32{0x00100293, 0x00100073}},
		{"fim de linha do Windows", "0x00100293\r\n0x00100073\r\n", []uint32{0x00100293, 0x00100073}},
		{"colado de um código Go", "\t0x00100293, // 0x00 addi x5, x0, 1\n\t0x00100073, // ebreak\n", []uint32{0x00100293, 0x00100073}},
		{"marcador do Bloco de Notas", "\ufeff0x00100293\n", []uint32{0x00100293}},
		{"sem quebra de linha no fim", "0x00100293", []uint32{0x00100293}},
	}

	for _, c := range casos {
		obtido, err := LerPalavras(strings.NewReader(c.texto))
		if err != nil {
			t.Errorf("%s: erro inesperado: %v", c.nome, err)
			continue
		}
		if !iguais(obtido, c.esperado) {
			t.Errorf("%s: obtido %#x, esperado %#x", c.nome, obtido, c.esperado)
		}
	}
}

// TestLerPalavrasErros confere o que é recusado e se a mensagem aponta a linha.
func TestLerPalavrasErros(t *testing.T) {
	casos := []struct {
		nome   string
		texto  string
		trecho string // pedaço que a mensagem de erro deve conter
	}{
		{"não é hexadecimal", "0x00100293\nabcdefgh\n", "linha 2"},
		{"só o prefixo", "0x\n", "linha 1"},
		{"mais de 8 dígitos", "0x001002931\n", "linha 1"},
		{"duas instruções na mesma linha", "0x00100293 0x00100073\n", "linha 1"},
		{"caractere inválido", "0x00100293\n0x00100073\n0x0010029G\n", "linha 3"},
		{"número negativo", "-1\n", "linha 1"},
		{"arquivo vazio", "", "nenhuma instrução"},
		{"só comentários", "// nada\n# nada\n\n", "nenhuma instrução"},
	}

	for _, c := range casos {
		_, err := LerPalavras(strings.NewReader(c.texto))
		if err == nil {
			t.Errorf("%s: deveria dar erro", c.nome)
			continue
		}
		if !strings.Contains(err.Error(), c.trecho) {
			t.Errorf("%s: erro = %q, esperado conter %q", c.nome, err.Error(), c.trecho)
		}
	}
}

// TestEscreverELer confere que o que se escreve é o que se lê de volta.
func TestEscreverELer(t *testing.T) {
	for _, programa := range Padrao() {
		var saida bytes.Buffer
		if err := EscreverPalavras(&saida, programa.Palavras); err != nil {
			t.Fatalf("%s: erro ao escrever: %v", programa.Nome, err)
		}

		lidas, err := LerPalavras(&saida)
		if err != nil {
			t.Fatalf("%s: erro ao ler: %v", programa.Nome, err)
		}
		if !iguais(lidas, programa.Palavras) {
			t.Errorf("%s: o programa lido é diferente do escrito", programa.Nome)
		}
	}
}

// TestLerArquivo confere o nome do programa e o conteúdo lido de um arquivo.
func TestLerArquivo(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "meu_benchmark.hex")
	if err := os.WriteFile(caminho, []byte("0x00100293\n0x00100073\n"), 0o644); err != nil {
		t.Fatalf("erro ao criar o arquivo: %v", err)
	}

	programa, err := LerArquivo(caminho)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if programa.Nome != "meu_benchmark" {
		t.Errorf("Nome = %q, esperado %q", programa.Nome, "meu_benchmark")
	}
	if !iguais(programa.Palavras, []uint32{0x00100293, 0x00100073}) {
		t.Errorf("Palavras = %#x", programa.Palavras)
	}
}

// TestLerArquivoComErros confere o arquivo que não existe e o com conteúdo inválido.
func TestLerArquivoComErros(t *testing.T) {
	dir := t.TempDir()

	if _, err := LerArquivo(filepath.Join(dir, "nao_existe.hex")); err == nil {
		t.Error("um arquivo que não existe deveria dar erro")
	}

	caminho := filepath.Join(dir, "invalido.hex")
	if err := os.WriteFile(caminho, []byte("0x00100293\nlixo\n"), 0o644); err != nil {
		t.Fatalf("erro ao criar o arquivo: %v", err)
	}
	_, err := LerArquivo(caminho)
	if err == nil {
		t.Fatal("um conteúdo inválido deveria dar erro")
	}
	if !strings.Contains(err.Error(), "invalido.hex") || !strings.Contains(err.Error(), "linha 2") {
		t.Errorf("erro = %q, esperado citar o arquivo e a linha 2", err.Error())
	}
}

// TestMedirProgramaDeArquivo confere o caminho completo: escrever um programa
// num arquivo, lê-lo e medi-lo, como a janela vai fazer com o benchmark do professor.
func TestMedirProgramaDeArquivo(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "alfabeto.hex")
	arquivo, err := os.Create(caminho)
	if err != nil {
		t.Fatalf("erro ao criar o arquivo: %v", err)
	}
	if err := EscreverPalavras(arquivo, programas.Alfabeto); err != nil {
		t.Fatalf("erro ao escrever: %v", err)
	}
	arquivo.Close()

	programa, err := LerArquivo(caminho)
	if err != nil {
		t.Fatalf("erro ao ler: %v", err)
	}

	r, err := Medir(context.Background(), programa, Opcoes{Repeticoes: 2})
	if err != nil {
		t.Fatalf("erro na medição: %v", err)
	}
	if r.Programa != "alfabeto" || r.Instrucoes != 57 || r.Registradores[8] != 0 {
		t.Errorf("Programa = %q, Instrucoes = %d, x8 = %d; esperado alfabeto, 57 e 0",
			r.Programa, r.Instrucoes, r.Registradores[8])
	}
	if !r.Passou() {
		t.Errorf("sem resultado esperado, não deveria haver falhas: %v", r.Falhas)
	}
}
