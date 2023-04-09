## Descrição
O objetivo desse projeto é ler a partir de uma fila do RabbitMQ as informações de um envio de email, se houver anexos pegar eles do Minio e enviar o e-mail formatado para um servidor SMTP

## Objetivos
- [x] Ler destinatário, descrição, mensagem e caminho de anexos do email a partir de uma fila do RabbitMQ
- [x] Fazer envio de e-mail sem anexo
- [x] Fazer envio de e-mail com anexo
- [x] Ler anexo do Minio
- [x] Criar cache local de anexos
- [x] Criar fila dos mortos, e-mails com mais X tentativas de envio

## Métricas
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
