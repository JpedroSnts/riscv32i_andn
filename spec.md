# Plano de implementação do simulador RISC-V

Este documento organiza as funcionalidades pendentes em etapas. A ordem começa por tornar confiável o caminho já existente (decodificação, ULA e execução), depois liga CPU, memória e dispositivos, e por fim trata os extras.

## Estado atual do repositório

- `cpu/ula.go` já implementa operações aritméticas e lógicas, incluindo `ANDN`.
- `cpu/decode.go` já decodifica instruções dos tipos R, I, S e B, mas ainda é preciso validar os imediatos e as combinações de campos com testes.
- `cpu/cpu.go` já executa algumas instruções R, I, `LW`, `SW` e desvios condicionais. Há um erro evidente no `SW`: o valor armazenado é lido de `Regs[inst]`, em vez do registrador indicado pelo campo `rs2`.
- `barramento/barramento.go` define somente a interface de leitura e escrita de palavras de 32 bits.
- `mem/ram.go` declara o tipo `RAM`, mas não implementa leitura ou escrita.
- `cmd/sim/main.go` apenas imprime o título do simulador; ainda não instancia a CPU nem executa um programa.
- Não há testes no repositório neste momento.

## 1. Corrigir e testar o núcleo existente

1. Corrija a execução de `SW` para selecionar o registrador fonte pelo campo `rs2` da instrução. Confirme que `x0` permanece sempre zero e que a escrita em `rd = x0` não altera estado observável.
2. Revise as funções `immI`, `immS` e `immB`: cada imediato deve reconstruir os bits definidos pela ISA, incluir todos os bits fragmentados do formato e fazer extensão de sinal na largura correta. Para branches, confira especialmente os bits das posições 11 e 12 e o alinhamento de dois bytes.
3. Crie testes unitários em `cpu` para cada operação da ULA, incluindo valores com sinal e sem sinal, deslocamentos de 0 e 31 posições e casos que diferenciem `AND` de `ANDN`.
4. Crie testes para a decodificação: campos `rd`, `rs1`, `rs2`, `funct3` e `funct7`; imediatos positivos e negativos; e combinações de funct que devem ser rejeitadas.
5. Crie testes de `CPU.Step` para instruções R e I já suportadas, `LW`, `SW`, os seis branches implementados, atualização sequencial do `PC` e tomada/não tomada de desvio. Use um barramento falso simples nos testes para observar as leituras e escritas.
6. Rode `go test ./...` após cada grupo de correções e mantenha os testes pequenos o bastante para apontarem qual comportamento falhou.

## 2. Fazer o primeiro programa executar de ponta a ponta

1. Implemente uma memória simples em `mem` para o primeiro experimento. Defina se os endereços recebidos são bytes ou índices de palavras; como a CPU soma imediatos a endereços de byte, a implementação deve converter o endereço de forma consistente ao acessar `[]uint32`.
2. Escolha e documente um comportamento para endereço fora da memória e acesso desalinhado (por exemplo, erro explícito ou política definida pelo projeto). Não deixe que um acesso inválido vire um panic acidental.
3. Em `cmd/sim/main.go`, instancie a memória, carregue um pequeno programa de teste, crie a CPU e execute instruções até um limite definido ou até uma condição de parada.
4. O programa de demonstração deve produzir um resultado verificável para `ANDN` e, em seguida, exercitar ao menos uma escrita e leitura de memória. Mostre os registradores ou os valores esperados no terminal.
5. Execute `go run ./cmd/sim` e compare a saída observada com os valores esperados. Mantenha esse caminho como demonstração mínima do projeto.

## 3. Implementar a RAM com o mapa de memória do enunciado

1. Recupere o mapa de endereços exato do enunciado da semana 4 antes de codificar esta etapa. O mapa não está incluído nos arquivos atuais; não invente endereços ou tamanhos. Registre no código/documentação os intervalos, tamanhos, permissões e o dispositivo associado a cada região.
2. Atualize `RAM` para representar a região de memória atribuída a ela, incluindo seu endereço inicial e capacidade, ou adote outra estrutura compatível com o mapa oficial.
3. Implemente as operações exigidas pelo barramento, traduzindo endereços de byte para posições internas e verificando limites e alinhamento.
4. Defina e implemente o comportamento para leituras e escritas inválidas de acordo com o contrato do projeto; se necessário, ajuste a interface para propagar erros em vez de ocultá-los.
5. Teste o primeiro e o último endereço válido, endereços fora da região, alinhamento, preservação dos dados e conversão entre endereço de byte e índice interno.

## 4. Completar e integrar o barramento

1. Use o mapa oficial para decidir quais regiões pertencem à RAM, à VRAM, ao teclado e a outros dispositivos previstos. A interface atual em `barramento` só especifica `Read32` e `Write32`; confirme se ela também precisa suportar bytes/meias-palavras, erros ou interrupções.
2. Implemente um barramento concreto que encaminhe cada operação ao dispositivo dono do endereço. Centralize a validação de regiões sobrepostas e de endereços sem dispositivo.
3. Garanta que a CPU continue dependendo da interface de barramento, e não de implementações concretas de RAM ou VRAM.
4. Adicione testes de roteamento: cada região deve atingir o dispositivo correto; endereços não mapeados e limites entre regiões devem seguir o comportamento definido.
5. Integre CPU, RAM e barramento no simulador e confirme que o programa da etapa 2 continua passando pelo barramento completo.

## 5. Implementar VRAM e exibição no terminal

1. Defina o formato da VRAM conforme o enunciado: endereço e tamanho da região, dimensões da tela, representação de cada célula/pixel e codificação de cores ou caracteres.
2. Implemente a VRAM como dispositivo do barramento, com leituras e escritas limitadas à sua região e conversão de endereço para coordenada.
3. Crie uma rotina de renderização que percorra a VRAM e imprima o conteúdo no terminal com dimensões e cores definidas pelo projeto. Separe a lógica de renderização da CPU para facilitar testes.
4. Decida quando atualizar a tela (por instrução, por quadro ou sob comando) e evite limpar/redesenhar o terminal sem necessidade. Documente essa decisão.
5. Teste a conversão de coordenadas, limites da tela e a saída de um quadro pequeno conhecido; depois valide visualmente a saída no simulador.

## 6. Criar o programa de teste do sistema

1. Escreva um programa curto que use somente instruções já implementadas e que possa ser carregado pelo simulador sem ferramentas externas. Identifique claramente o formato das instruções e os endereços usados.
2. Inclua no programa operações aritméticas/lógicas (especialmente `ANDN`), desvios, `LW`/`SW` e ao menos uma escrita na VRAM.
3. Defina o estado final esperado: registradores relevantes, memória e conteúdo exibido. Use-o como teste de integração reproduzível.
4. Se o programa exigir uma instrução ainda não suportada, implemente e teste essa instrução antes de adicioná-la ao programa; não masqueie instruções inválidas como operações bem-sucedidas.

## 7. Completar o conjunto de instruções necessário

1. Compare o escopo pedido no enunciado com os opcodes realmente tratados em `CPU.Step`. O código atual cobre apenas parte do conjunto RV32I; não se deve considerar a ISA completa até verificar cada instrução exigida.
2. Para cada instrução pendente, implemente primeiro a decodificação e o imediato, depois a semântica da CPU e por fim os testes unitários e de integração correspondentes.
3. Trate instruções inválidas de forma explícita e consistente, sem avançar silenciosamente como se tivessem sido executadas corretamente.
4. Atualize o programa de teste somente depois que a instrução necessária passar nos testes isolados.

## 8. Extras: interrupção por teclado e interface interativa

Estas etapas são opcionais e devem começar depois de CPU, barramento e programa de teste estarem estáveis.

1. Consulte o enunciado para determinar o endereço e o protocolo do teclado, o significado de uma interrupção e como a CPU deve reconhecê-la. A arquitetura RV32I básica, sozinha, não define o dispositivo de teclado deste projeto.
2. Modele o teclado como dispositivo do barramento e defina como uma tecla pressionada fica disponível ao programa (por registrador mapeado em memória, fila de eventos ou mecanismo especificado).
3. Defina o contrato de interrupção antes de implementá-la: estado salvo, ponto de atendimento, rotina de serviço, retorno e comportamento quando não há interrupção pendente. Se o projeto exigir registradores de controle e status, inclua-os conforme o contrato definido.
4. Teste a interrupção sem terminal primeiro, usando eventos de teclado simulados e verificando que a execução normal é retomada corretamente.
5. Só então conecte a captura de entrada do terminal e a renderização da VRAM. Mantenha a CPU executando em um fluxo controlado, permita encerramento limpo e evite que entrada, execução e renderização corrompam umas às outras.
6. Valide o fluxo completo manualmente e preserve testes automatizados para o protocolo do dispositivo e para o tratamento da interrupção.

## Critérios de conclusão

- `go test ./...` passa e cobre os comportamentos implementados.
- `go run ./cmd/sim` executa o programa de teste de forma reproduzível e apresenta os resultados esperados.
- O mapa de memória e o protocolo dos dispositivos correspondem ao enunciado oficial e estão documentados.
- Funcionalidades opcionais só são marcadas como concluídas quando têm teste automatizado e uma validação integrada no simulador.