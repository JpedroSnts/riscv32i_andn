# Tarefas do simulador RISC-V

Checklist derivado de [spec.md](spec.md) e [design.md](design.md). Execute as fases em ordem; tarefas de VRAM, mapa definitivo e interrupção dependem de informações do enunciado oficial.

## Estado inicial

- A ULA já contém operações aritméticas/lógicas, incluindo `ANDN`; decodificação e execução já cobrem parte das instruções.
- RAM, barramento concreto, programa executável e testes ainda precisam ser implementados.
- A implementação atual de `SW` usa o índice errado para ler o registrador fonte.
- O mapa de endereços, o formato da VRAM e o protocolo do teclado não estão nos arquivos do repositório.

## Fase 0: confirmar requisitos externos

- [ ] Obter o enunciado oficial da semana 4 e registrar endereços iniciais, tamanhos, permissões e dispositivos de cada região.
- [ ] Confirmar quais instruções RV32I são obrigatórias e se acessos de byte/meia-palavra (`LB`, `LBU`, `LH`, `LHU`, `SB`, `SH`) fazem parte do escopo.
- [ ] Confirmar formato da VRAM: dimensões, codificação de célula/pixel, tamanho e modo de atualização.
- [ ] Confirmar, se os extras forem exigidos, protocolo do teclado e semântica de interrupção: reconhecimento, estado salvo, rotina de serviço e retorno.

**Saída da fase:** requisitos documentados sem endereços ou comportamentos presumidos.

## Fase 1: estabilizar CPU e decodificação

- [ ] Corrigir `SW` para armazenar o valor de `Regs[rs2(inst)]`.
- [ ] Garantir que `x0` sempre leia como zero e ignore escritas.
- [ ] Revisar e corrigir a reconstrução e extensão de sinal de `immI`, `immS` e `immB`, incluindo todos os bits fragmentados do branch.
- [ ] Definir como instruções inválidas ou não suportadas são reportadas; impedir avanço silencioso do `PC` em instruções inválidas.
- [ ] Definir o contrato de erros de `CPU.Step` e do barramento antes de integrar dispositivos; propagar falhas sem deixar estado da CPU parcialmente atualizado.
- [ ] Adicionar testes da ULA, incluindo operações com e sem sinal, shifts de 0 e 31 bits e casos que distingam `ANDN` de `AND`.
- [ ] Adicionar testes de campos de instrução, imediatos positivos/negativos e combinações inválidas de funct.
- [ ] Adicionar testes de `CPU.Step` para instruções implementadas, `LW`, `SW`, os seis branches, atualização do `PC`, `x0` e erros; usar barramento falso.
- [ ] Rodar `go test ./...` e corrigir falhas relacionadas a esta fase.

**Aceite:** testes isolados validam decodificação, operações e execução de cada instrução atualmente suportada.

## Fase 2: RAM simples e demonstração mínima

- [ ] Definir uma memória de demonstração com endereços de byte e política explícita para desalinhamento e endereço fora dos limites.
- [ ] Implementar leitura/escrita na RAM sem permitir panic acidental em acessos inválidos.
- [ ] Criar testes de RAM para primeiro/último endereço válido, limites, alinhamento, preservação de dados e conversão de endereço.
- [ ] Montar em `cmd/sim` uma CPU com RAM e carregar um programa curto em palavras de instrução.
- [ ] Executar `ANDN`, gravar o resultado com `SW` e recuperá-lo com `LW`; limitar a execução por quantidade de instruções ou condição de parada.
- [ ] Exibir valores esperados no terminal e validar o fluxo com `go run ./cmd/sim`.

**Aceite:** a demonstração termina de forma determinística e mostra resultado de `ANDN` e memória conferidos.

## Fase 3: mapa de memória e barramento

- [ ] Transcrever para a documentação a tabela de regiões confirmada no enunciado oficial.
- [ ] Definir o contrato do barramento para as larguras de acesso exigidas; ampliar a interface atual, se forem necessários acessos subpalavra ou erros explícitos.
- [ ] Implementar o barramento concreto com registro de regiões e encaminhamento para o dispositivo correto.
- [ ] Rejeitar regiões sobrepostas na configuração e reportar acessos não mapeados ou fora do intervalo.
- [ ] Implementar RAM com endereço base, capacidade e permissões definidos pelo mapa oficial.
- [ ] Adicionar testes de roteamento, limites entre regiões, sobreposição e endereço sem dispositivo.
- [ ] Integrar CPU, RAM e barramento concreto no simulador e repetir a demonstração da fase 2.

**Aceite:** todos os acessos passam pelo barramento e chegam ao único dispositivo dono do endereço.

## Fase 4: VRAM e saída no terminal

- [ ] Registrar no projeto os dados oficiais da VRAM: base, tamanho, dimensões, codificação e acessos permitidos.
- [ ] Implementar VRAM como dispositivo do barramento, com validação de limites e conversão de endereço para célula/pixel.
- [ ] Implementar renderização separada da CPU que converte um quadro conhecido em saída de terminal.
- [ ] Definir quando a tela é atualizada e evitar redesenhos desnecessários.
- [ ] Testar conversão, limites e quadro conhecido; validar visualmente a saída no simulador.

**Aceite:** uma escrita da CPU na VRAM produz a célula/pixel esperado, comprovado por teste e execução integrada.

## Fase 5: programa de integração e cobertura ISA

- [ ] Escrever um programa curto e determinístico, sem depender de assembler externo, e documentar suas palavras/endereço inicial.
- [ ] Cobrir `ANDN`, operações necessárias, desvio, `LW`/`SW` e ao menos uma escrita na VRAM.
- [ ] Definir e testar o estado final esperado: registradores relevantes, memória e conteúdo exibido.
- [ ] Comparar as instruções exigidas no enunciado com as implementadas no `CPU.Step`.
- [ ] Implementar cada instrução pendente necessária: decodificação/imediato, semântica da CPU, teste unitário e inclusão no programa de integração.
- [ ] Rodar `go test ./...` e `go run ./cmd/sim` após integrar o programa completo.

**Aceite:** o programa completo produz o estado final documentado e não executa silenciosamente instruções inválidas.

## Fase 6: extras opcionais

- [ ] Implementar o dispositivo de teclado segundo o mapa e protocolo oficiais.
- [ ] Definir a fila/evento de entrada e o ponto em que a CPU reconhece uma interrupção.
- [ ] Implementar salvamento/restauração do estado e retorno da rotina conforme o contrato do projeto.
- [ ] Testar teclado e interrupção sem terminal, usando eventos simulados.
- [ ] Integrar captura de entrada, execução e renderização com encerramento controlado.
- [ ] Validar manualmente o fluxo interativo e manter testes automatizados que não dependam de terminal.

**Aceite:** os extras passam nos testes de dispositivo/interrupção e funcionam junto com o simulador sem tornar os testes básicos interativos.

## Verificação final

- [ ] `go test ./...` passa.
- [ ] `go run ./cmd/sim` executa o programa de integração com saída reproduzível.
- [ ] Mapa de memória e protocolos de dispositivos conferem com o enunciado oficial.
- [ ] Cada funcionalidade opcional concluída tem teste automatizado e validação integrada.
