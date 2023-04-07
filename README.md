## Descrição
O objetivo desse projeto é ler a partir de uma fila do RabbitMQ as informações de um envio de email, se houver anexos pegar eles do Minio e enviar o e-mail formatado para um servidor SMTP

## Objetivos
- [x] Ler destinatário, descrição, mensagem e caminho de anexos do email a partir de uma fila do RabbitMQ
- [x] Fazer envio de email sem anexo
- [ ] Fazer envio de email com anexo
- [ ] Ler anexo do Minio
- [ ] Criar cache do Minio

## Métricas
- [x] Expor as métricas do servidor na porta `8001`
- [ ] Pegar as seguintes métricas:
  - [x] Quantidade de e-mails recebidos da fila do RabbitMQ
  - [x] Quantidade de bytes recebidos da fila do RabbitMQ
  - [x] Quantidade de e-mails enviados com sucesso
  - [x] Quantidade de bytes enviados no corpo do email
  - [x] Quantidade de e-mails reenviados para a fila
  - [x] Tempo de envio por lote de e-mails
  - [ ] Quantidade de e-mails enviados com anexo
  - [ ] Quantidade de anexos enviados 
  - [ ] Quantidade de bytes enviados no anexo
  - [ ] Tamanho do cache local do minio
