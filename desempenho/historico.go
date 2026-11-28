package desempenho

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// ArquivoPadrao é o histórico usado quando ninguém escolhe outro. É um caminho relativo: fica na pasta em que o programa foi executado, que é a raiz do projeto quando se roda com go run a partir dela.
const ArquivoPadrao = "historico.json"

// Registro é uma medição guardada, com um rótulo que identifica a versão do código (por exemplo "linha de base" ou "depois de tirar a trava").
type Registro struct {
	Rotulo    string    `json:"rotulo"`
	Data      time.Time `json:"data"`
	Resultado Resultado `json:"resultado"`
}

// Carregar lê todas as medições de um arquivo de histórico.
// Um arquivo que não existe ou está vazio é um histórico vazio, e não um erro.
func Carregar(caminho string) ([]Registro, error) {
	dados, err := os.ReadFile(caminho)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(dados)) == 0 {
		return nil, nil
	}

	var registros []Registro
	if err := json.Unmarshal(dados, &registros); err != nil {
		return nil, fmt.Errorf("%s: histórico inválido: %w", caminho, err)
	}
	return registros, nil
}

// Salvar acrescenta uma medição ao arquivo de histórico. Se o registro não tiver data, usa a hora atual. Um arquivo que não dá para ler nunca é sobrescrito: nesse caso devolve erro e não grava nada.
func Salvar(caminho string, registro Registro) error {
	if registro.Data.IsZero() {
		registro.Data = time.Now()
	}

	registros, err := Carregar(caminho)
	if err != nil {
		return err
	}
	registros = append(registros, registro)

	dados, err := json.MarshalIndent(registros, "", "  ")
	if err != nil {
		return err
	}

	// escreve num arquivo temporário e só depois troca, para uma falha no meio da gravação não destruir o histórico
	temporario := caminho + ".tmp"
	if err := os.WriteFile(temporario, dados, 0o644); err != nil {
		return err
	}
	return os.Rename(temporario, caminho)
}

// UltimoDoMesmoPrograma devolve o registro mais recente medido com o mesmo programa (mesmo nome e mesma quantidade de instruções), e false se não houver.
func UltimoDoMesmoPrograma(registros []Registro, r Resultado) (Registro, bool) {
	for i := len(registros) - 1; i >= 0; i-- {
		anterior := registros[i].Resultado
		if anterior.Programa == r.Programa && anterior.Instrucoes == r.Instrucoes {
			return registros[i], true
		}
	}
	return Registro{}, false
}
