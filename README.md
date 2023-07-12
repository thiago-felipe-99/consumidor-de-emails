## Descrição
O objetivo desse projeto é ler a partir de uma fila do RabbitMQ as informações de um envio de email, se houver anexos pegar eles do Minio e enviar o e-mail formatado para um servidor SMTP.

Para isso foi criado 2 módulos:
- Publisher, a partir de uma API HTTP ele envia os emails para a fila do RabbitMQ
- Consumer, ele faz o envio dos emails da fila do RabbitMQ para um servidor SMTP

## Como Rodar
Para iniciar os módulos é necessário criar um arquivo `.env` na raiz do projeto, tem um exemplo de `.env` no arquivo `env_example`, para um teste rápido pode mudar só as linha com `#CHANGE_ME`.

Com o arquivo `.env` criado inicie os containers com:
```shell 
docker compose up -d
```
Ele irá iniciar todos os containers do projeto, assim podemos ver todos os caminhos da api HTTP em [`http://localhost:8080/swagger`](http://localhost:8080/swagger).

### Como Rodar Em Ambiente de Desenvolvimento
Para rodar de uma forma mais rápida o Publisher e o Consumer podemos subir todos os continaers estáticos:
```shell
docker compose up rabbit minio prometheus grafana database createbuckets -d
```
E rodar separadamente o Publisher com: 
```shell
make run_publisher
```
Ou/e rodar separadamente o Consumer com: 
```shell
make run_consumer
```

Para rodar os lints do projeto basta executar:
```shell
make all
```

## Objetivos Publisher
- [x] Criar sistema de usuários
- [x] Criar mecanismos de autenticação
- [x] Fazer envio de emails
- [x] Fazer envio de anexos
- [x] Criar sistema de template de emails 
- [x] Criar sistema para gerenciar filas no RabbitMQ
- [x] Criar sistema para gerenciar listas de emails
- [x] Adicionar Swagger na API 

## Objetivos Consumer
- [x] Ler destinatário, descrição, mensagem e caminho de anexos do email a partir de uma fila do RabbitMQ
- [x] Fazer envio de e-mail sem anexo
- [x] Fazer envio de e-mail com anexo
- [x] Ler anexo do Minio
- [x] Criar cache local de anexos
- [x] Criar fila dos mortos, e-mails com mais X tentativas de envio

## Métricas Publisher
- [x] Expor as métricas no caminho `/metrics`
- [x] Pegar métrica de uso por caminho da API

## Métricas Consumer
- [x] Expor as métricas do servidor na porta `8001`
- [x] Pegar as seguintes métricas:
  - [x] Quantidade de e-mails recebidos da fila do RabbitMQ
  - [x] Quantidade de bytes recebidos da fila do RabbitMQ
  - [x] Quantidade de e-mails enviados com sucesso
  - [x] Quantidade de bytes enviados no corpo do email
  - [x] Quantidade de e-mails reenviados para a fila
  - [x] Quantidade de e-mails enviados para a fila dos mortos
  - [x] Quantidade de e-mails enviados com anexo
  - [x] Quantidade de anexos enviados 
  - [x] Quantidade de bytes enviados no anexo
  - [x] Quantidade de anexos no cache local
  - [x] Quantidade de bytes no cache local
  - [x] Tempo de envio por lote de e-mails

