# 🚨 Sistema de Gerenciamento de Alarmes

Bem-vindo ao repositório do **Sistema de Alarme**! Este projeto foi desenvolvido como uma solução para gerenciamento de eventos de sensores, utilizando uma arquitetura moderna de microsserviços e comunicação assíncrona.

<p align="center">
  <img src="docs/capa.jpg" alt="Sistema de Gerenciamento de Alarmes" width="800">
</p>

## 🎯 Objetivo
O objetivo principal desta aplicação é processar eventos de dispositivos IoT (como sensores de movimento) de forma escalável e resiliente. Todo o fluxo foi desenhado para evitar o acoplamento direto entre a captura do evento e a geração do alarme, utilizando mensageria para garantir a entrega e o processamento confiável.

## 🏗️ Visão Geral da Arquitetura

O ecossistema foi projetado de forma orientada a eventos e é composto pelos seguintes blocos:

*   **📥 Event Service:** A porta de entrada do sistema. É uma API HTTP leve que recebe os eventos disparados pelos dispositivos e atua unicamente como um *Publisher*, enviando a carga de dados imediatamente para o broker.
*   **📨 Broker de Mensagens:** O intermediário responsável pela comunicação assíncrona. Ele recebe os eventos do `Event Service` e os enfileira com segurança, garantindo que o sistema produtor não precise realizar chamadas HTTP diretas para o sistema consumidor.
*   **🚨 Alarm Service:** O *Consumer* da arquitetura. Ele escuta ativamente o broker e, ao identificar um evento específico (como `MOTION_DETECTED`), processa a regra de negócio, cria o alarme e o persiste no banco de dados. Além disso, ele expõe as APIs para listagem (`GET /alarms`) e para o desligamento/encerramento de alarmes disparados (`PATCH /alarms/{id}/close`).
*   **🗄️ Banco de Dados:** Onde o estado final dos alarmes é persistido em uma estrutura relacional.

---

## 🛠️ Tecnologias Utilizadas (Stack)

O ecossistema do projeto foi construído utilizando as ferramentas mais adequadas para garantir performance, escalabilidade e facilidade de manutenção. 

**Back-end:**
*   **Go (Golang):** Linguagem escolhida para o `event-service` e `alarm-service`. Sua alta performance, baixo consumo de memória e excelente modelo de concorrência nativo (goroutines/channels) a tornam perfeita para sistemas assíncronos e processamento de filas.
*   **Swagger / OpenAPI:** Utilizado para a documentação clara, padronizada e interativa dos endpoints da API REST.

**Mensageria e Banco de Dados:**
*   **Eclipse Mosquitto (MQTT):** Broker de mensagens extremamente leve, que utiliza o protocolo padrão da indústria para IoT. Excelente para receber alto volume de eventos de sensores com baixa latência.
*   **PostgreSQL:** Banco de dados relacional robusto, confiável e com total conformidade ACID, garantindo a integridade dos registros de alarmes e seus status.

**Front-end (Painel de Operação):**
*   **React + TypeScript (TSX):** Biblioteca base para a construção da interface do usuário de forma componentizada, com o bônus da tipagem estática para evitar erros em tempo de desenvolvimento.
*   **Vite:** Ferramenta de build de altíssima velocidade, proporcionando um ambiente de desenvolvimento ágil e gerando *bundles* otimizados para produção.
*   **Tailwind CSS:** Framework CSS utility-first, utilizado para construir a interface visual com temática de painel industrial diretamente no código dos componentes.

**Infraestrutura:**
*   **Docker & Docker Compose:** Responsáveis pela conteinerização de todas as peças (front, back, broker e banco). Garantem que a aplicação rode de maneira idêntica em qualquer ambiente, com um único comando.

---

## 🚀 Como executar o projeto

Todo o ambiente foi conteinerizado para facilitar a execução, sem a necessidade de instalar dependências locais na sua máquina (como Go, Node ou bancos de dados).

1. Clone este repositório:
   ```
   git clone [https://github.com/BrunoBerval/alarm_system_test.git](https://github.com/BrunoBerval/alarm_system_test.git)
   cd alarm_system_test
   ```
2. Suba a infraestrutura completa utilizando o Docker Compose:
    ```
        docker compose up -d --build
    ```

Isso fará o download das imagens, compilará o código e iniciará os seguintes serviços simultaneamente:

- **Broker MQTT** (Mosquitto) na porta `1883`
- **Banco de Dados** (PostgreSQL) na porta `5432`
- **Event Service** (API HTTP de entrada) na porta `8080`
- **Alarm Service** (API HTTP de leitura e persistência) na porta `8081`
- **Painel de Operação** (Client App Web) na porta `3000`

Após o terminal indicar que os containers estão rodando, o projeto estará pronto para uso.

---

## 🧪 Como testar a API

Você pode interagir com o sistema de três formas: pelo Painel de Operação Web, pela documentação interativa do Swagger ou via linha de comando com `curl`.

### 1. Painel de Operação (Web)

Acesse no seu navegador: `http://localhost:3000`
A interface gráfica permite disparar eventos (individuais ou em lote) e acompanhar o status dos alarmes em tempo real.

### 2. Swagger UI (OpenAPI)

A documentação interativa da API está disponível via Swagger no Alarm Service. Acesse no seu navegador:
`http://localhost:8081/swagger/index.html`

### 3. Via cURL (Terminal)

**Disparar um evento (Event Service):**

```bash
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{"device_id": "sensor-001", "type": "MOTION_DETECTED"}'
```

---
## 🧠 Decisões Técnicas

### 1. Stack
**Qual linguagem, framework e bibliotecas você escolheu? Por quê? Explique os principais motivos da escolha e quais alternativas considerou.**

Para a construção deste sistema, optei por uma stack moderna e de alta performance, separada em services e client:

**services:**
*   **Linguagem:** Go (Golang).
*   **Bibliotecas:** Biblioteca padrão `net/http` (aproveitando o roteamento aprimorado do Go 1.22+), `log/slog` para logs estruturados, e `paho.mqtt.golang` para comunicação com o broker.
*   **Por quê?** Go é muito eficiente em microsserviços e sistemas orientados a eventos por conta de seu modelo de concorrência nativo (goroutines e channels) o que facilitou a criação de um *Worker Pool* seguro e performático no `event-service` para enfileirar e despachar eventos sem bloquear a API HTTP. A compilação para um binário estático único resulta em containers Docker extremamente leves (baseados em Alpine) de inicialização quase instantânea. Cheguei a considerar utilizar Python devido a facilidade de escrita graças a sua pouca verbosidade, porém Go oferece um desempenho superior. C++ entregaria um processamento maior ainda, mas como não possuo muita intimidade com a linguagem foi descartada, além de que gestão manual de memória e configurações complexas de bibliotecas externas seriam overkill para este projeto.

**client:**
*   **Linguagem/Framework:** React com TypeScript, empacotado via Vite.
*   **Estilização:** Tailwind CSS.
*   **Por quê?** React facilita o gerenciamento de estado da interface (como o cálculo de eventos em lote e o status luminoso dos alarmes). Vite proporciona um ambiente de desenvolvimento ultra-rápido. Tailwind CSS permitiu estilizar o "painel industrial" diretamente nos componentes de forma ágil, utilizando variáveis CSS customizadas.

### 2. Broker
**Por que você escolheu MQTT ou Kafka? Explique por que considera a tecnologia escolhida adequada para esse cenário e quais seriam as principais vantagens e desvantagens da outra opção.**

Decidi optar por utilizar MQTT (via Eclipse Mosquitto) pois o cenário do teste propõe o gerenciamento de eventos emitidos por dispositivos físicos (sensores). O MQTT foi criado especificamente para a Internet das Coisas (IoT). Ele é um protocolo extremamente leve, com baixíssimo *overhead* de cabeçalho, projetado para garantir a entrega de mensagens mesmo em redes instáveis ou com largura de banda restrita. Para o tráfego de *payloads* pequenos (como o JSON simples contendo apenas a identificação e o tipo do sensor), o MQTT oferece uma latência baixíssima. Além disso, o broker Mosquitto é incrivelmente leve e fácil de conteinerizar, consumindo poucos megabytes de RAM.

Havia a opção de utilizar Apache Kafka que é uma plataforma robusta de *streaming* distribuído com altíssimo *throughput*. Sua maior vantagem seria a durabilidade dos eventos (o Kafka grava as mensagens em disco), o que permite o *replay* de eventos antigos caso um consumidor caia ou precise reprocessar o histórico. Mas a sua complexidade e o custo operacional fizeram com que se tornasse uma alternativa menos interessante para esta tarefa. "Não se deve usar um canhão para caçar coelhos".

### 3. Banco de dados
**Por que escolheu esse banco de dados? Explique brevemente sua decisão.**

A escolha do PostgreSQL que é uma ferramenta bastante utilizada no mercado, se baseou na necessidade de um banco de dados relacional robusto, open-source. A estrutura dos alarmes propostos possui um esquema de dados bem definido e estruturado (`id`, `device_id`, `type`, `status`, `created_at`), o que se encaixa perfeitamente no modelo relacional.

### 4. Microserviços
**Por que separar o sistema em dois serviços? O que você considera como vantagens e desvantagens dessa abordagem?**

A separação do sistema no `Event Service` (porta de entrada) e no `Alarm Service` (processamento e armazenamento) implementa o princípio da separação de responsabilidades, fundamental em arquiteturas orientadas a eventos. Essa uma abordagem que possui várias, como: 

Vantagens da abordagem:
*   **Escalabilidade Independente:** Sensores de IoT podem gerar picos massivos de requisições. Separando os serviços, podemos escalar horizontalmente apenas o `Event Service` para dar conta do volume de entrada, sem precisar escalar junto o banco de dados e as regras de negócio.
*   **Resiliência e Tolerância a Falhas:** O broker atua como um "amortecedor". Se o `Alarm Service` ou o banco de dados ficarem offline para manutenção ou caírem, o `Event Service` não é afetado. Ele continua aceitando requisições e enfileirando os dados no broker. Quando o consumidor voltar, ele processa o passivo sem perda de dados.
*   **Desacoplamento Tecnológico:** Permite que no futuro os serviços sejam reescritos em linguagens diferentes ou mantidos por equipes separadas sem impactos mútuos. 

Mas também possui desvantagens: 
*   **Complexidade Operacional e de Infraestrutura:** Requer orquestração de rede, configuração do broker (MQTT) e gerenciamento de múltiplos containers, o que é nativamente mais complexo do que rodar um monólito simples.
*   **Consistência Eventual:** O fluxo deixa de ser síncrono. O cliente HTTP que enviou o evento não recebe a confirmação imediata de que o alarme foi salvo no banco, apenas de que o evento foi aceito. Existe uma micro-latência até o processamento ser concluído pelo `Alarm Service`.
*   **Dificuldade de Rastreabilidade:** Fazer o *debug* (troubleshooting) de um evento que falhou exige analisar logs em três lugares diferentes (Event Service, Broker e Alarm Service), demandando boas práticas de logs estruturados e IDs de correlação.

### 5. Falha no processamento
**Imagine que o Alarm Service receba uma mensagem, mas ocorra um erro ao salvar o alarme no banco.**
**O que deveria acontecer com essa mensagem?**

O que deveria acontecer é a mensagem não pode ser descartada silenciosamente nem
reprocessada infinitamente. Em um sistema de alarmes, perder um evento significa não
gerar um alarme que deveria existir, é uma falha de segurança. A estratégia adotada foi fazer um retry com backoff + DLQ.
Onde o fluxo era o seguinte: Ao falhar o `CreateAlarm`, o consumidor tenta novamente até 3 vezes, com espera crescente entre elas. Isso cobre a maioria das falhas reais, que são transitórias (banco reiniciando, timeout, conexão derrubada). Esgotadas as tentativas, o payload original é publicado no tópico `alarms/events/dlq`, saindo do fluxo principal mas permanecendo disponível para diagnóstico e reprocessamento.
Nesse caso o controle ficou na aplicação porque o MQTT não tem DLQ nativa, é somente publish e subscribe.


### 6. Mensagens duplicadas
**Imagine que o mesmo evento seja recebido duas vezes.**
**Como você evitaria a criação de dois alarmes para o mesmo evento? Não é necessário implementar uma solução complexa. Queremos entender seu raciocínio.**

As mensagens duplicadas foram tratadas, nessa aplicação a deduplicação não é feita por ID de mensagem, e sim pelo **estado do domínio**: antes de criar um alarme, o consumidor verifica se aquele dispositivo já possui um alarme com status `OPEN`.
Se já existe, o evento é ignorado e apenas registrado em log. A solução é elegante porque resolve dois problemas com a mesma regra: garante idempotência em reentregas do broker (QoS 1 é *at-least-once*) e, ao mesmo tempo, evita o acúmulo de alarmes redundantes quando um sensor dispara várias vezes durante a mesma ocorrência. Um novo alarme só volta a ser criado depois que o anterior for fechado via `PATCH /alarms/{id}/close` — o que é exatamente o comportamento esperado de um sistema de alarmes.

### 7. Evolução
**Imagine que, futuramente, o sistema precise enviar uma notificação sempre que um novo alarme for criado.**
**Como você adicionaria essa funcionalidade sem precisar alterar diretamente o Event Service**

O Event Service não precisaria mudar em nenhum cenário, porque ele não conhece quem consome suas mensagens — apenas publica no broker. Essa é justamente a vantagem do pub/sub: adicionar consumidores é uma operação aditiva. A abordagem escolhida seria criar um **Notification Service** que consome um novo tópico `alarms/created`, publicado pelo Alarm Service **após a persistência bem-sucedida** do alarme

## Diferenciais Implementados

- [x] Interface Web
- [x] Testes automatizados
- [x] Docker health checks
- [x] Retry de mensagens
- [x] Tratamento de mensagens duplicadas
- [x] Swagger/OpenAPI
- [x] Logs estruturados
- [x] Boa organização dos commits