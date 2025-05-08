# Projeto gocambio

## Descrição do Projeto

`gocambio` é uma aplicação cliente-servidor desenvolvida em Go como parte de um desafio técnico. O sistema consiste em:

* **Servidor (`server.go`):** Uma API HTTP que, ao receber uma requisição, busca a cotação atual do Dólar Americano (USD) para Real Brasileiro (BRL) de uma API externa. A cotação obtida é então registrada em um banco de dados SQLite e retornada ao cliente.
* **Cliente (`client.go`):** Uma aplicação que consome a API do servidor local para obter a cotação do dólar e salva essa cotação em um arquivo de texto.

Ambas as aplicações utilizam o pacote `context` do Go para gerenciar timeouts em suas operações críticas (chamada à API externa, persistência no banco de dados e requisição ao servidor). O projeto é totalmente dockerizado para facilitar a execução e o ambiente de desenvolvimento.

## Tecnologias Utilizadas

* **Linguagem:** Go (versão 1.24 ou conforme `go.mod`)
* **Banco de Dados:** SQLite
* **Conteinerização:** Docker & Docker Compose
* **Comunicação:** HTTP (Cliente-Servidor)
* **Parsing:** JSON

## Pré-requisitos

Para executar este projeto, você precisará ter instalado em sua máquina:

* Docker ([Instruções de Instalação](https://docs.docker.com/get-docker/))
* Docker Compose ([Instruções de Instalação](https://docs.docker.com/compose/install/))
* Git (para clonar o repositório)

## Estrutura do Projeto

```bash
./
├── client
│  ├── client.go
│  └── Dockerfile
├── docker-compose.yml
├── go.mod
├── README.md
└── server
  ├── Dockerfile
  └── server.go
```

## Configuração Inicial

1.  **Clone o repositório:**
    Se o projeto estiver no GitHub (exemplo com seu usuário):
    ```bash
    git clone [https://github.com/marcelofabianov/gocambio.git](https://github.com/marcelofabianov/gocambio.git)
    cd gocambio
    ```
    Caso contrário, certifique-se de que todos os arquivos listados na estrutura acima estejam no diretório `gocambio/`.

2.  **Módulos Go:**
    Os arquivos `go.mod` e `go.sum` já devem estar presentes e configurados. O processo de build do Docker (`docker-compose build`) irá executar `go mod download` dentro dos contêineres para baixar as dependências necessárias.

## Como Executar o Projeto com Docker Compose

Com o Docker e Docker Compose devidamente instalados, siga os passos abaixo na raiz do projeto (`gocambio/`):

1.  **Construa as imagens Docker e inicie os contêineres:**
    Execute o seguinte comando no seu terminal:
    ```bash
    docker-compose up --build
    ```
    * A flag `--build` garante que as imagens Docker sejam (re)construídas a partir dos `Dockerfiles` e do código fonte mais recente.
    * O serviço do servidor (`server`) será iniciado e ficará escutando na porta `8080` do seu host.
    * O serviço do cliente (`client`) será iniciado, fará uma requisição ao servidor para obter a cotação e, em seguida, salvará o resultado em um arquivo. Após sua tarefa, o contêiner do cliente será finalizado, mas o do servidor continuará ativo.

2.  **Verificando a Saída:**
    * **Logs:** Os logs do cliente e do servidor serão exibidos no terminal onde você executou o comando `docker-compose up`. Preste atenção a eles para verificar timeouts ou outros erros.
    * **Arquivo de Cotação do Cliente:** O cliente salvará a cotação do dólar no arquivo `gocambio/client_output/cotacao.txt` (relativo à raiz do projeto no seu sistema de arquivos local). O conteúdo esperado é:
        ```
        Dólar: X.YYYY
        ```
      (onde `X.YYYY` é o valor da cotação).
    * **Banco de Dados do Servidor:** O servidor registrará cada cotação obtida no banco de dados SQLite localizado em `gocambio/server_data/cotacoes.db` (relativo à raiz do projeto no seu sistema de arquivos local). Você pode utilizar qualquer ferramenta de visualização de SQLite para inspecionar os dados nesta tabela.

## Parando o Projeto

1.  Para parar os serviços (principalmente o servidor, que fica em execução), pressione `Ctrl+C` no terminal onde o `docker-compose up` está rodando.
2.  Após isso, para remover os contêineres criados pelo Docker Compose, execute:
    ```bash
    docker-compose down
    ```
3.  Se você desejar remover também os volumes nomeados (o que inclui os arquivos em `client_output/` e `server_data/` que foram criados pela execução do Docker Compose), utilize:
    ```bash
    docker-compose down -v
    ```

## Detalhes da Implementação do Desafio

* **Servidor (`server.go`):**
    * Ouve na porta `8080` no endpoint `/cotacao`.
    * Ao receber uma requisição `GET` em `/cotacao`:
        * Consulta a API externa `https://economia.awesomeapi.com.br/json/last/USD-BRL` com um timeout de **200 milissegundos**.
        * Persiste a cotação completa recebida (campo `bid` e outros detalhes) em um banco de dados SQLite (`cotacoes.db`) com um timeout de **10 milissegundos** para a operação de escrita.
        * Retorna o JSON completo obtido da API externa para o cliente.

* **Cliente (`client.go`):**
    * Realiza uma requisição HTTP `GET` para o endpoint `/cotacao` do servidor.
    * A URL do servidor é configurada através da variável de ambiente `SERVER_URL`. No `docker-compose.yml`, esta URL é definida como `http://server:8080/cotacao`, onde `server` é o nome do serviço do servidor na rede Docker.
    * Possui um timeout total de **300 milissegundos** para receber a resposta do servidor.
    * Extrai apenas o valor do campo `bid` (cotação de compra) do JSON retornado pelo servidor.
    * Salva a cotação no arquivo `cotacao.txt` no formato: `Dólar: {valor}`.

---