## Descrição
O objetivo desse projeto é ler a partir de uma fila do RabbitMQ as informações de um envio de email, se houver anexos pegar eles do Minio e enviar o e-mail formatado para um servidor SMTP

## Objetivos
- [x] Ler destinatário, descrição, mensagem e caminho de anexos do email a partir de uma fila do RabbitMQ
- [x] Fazer envio de email sem anexo
- [ ] Fazer envio de email com anexo
- [ ] Ler anexo do Minio
- [ ] Criar cache do Minio

## Metricas
- [x] Expor as metricas do servidor na porta `8001`
- [ ] Pegar as seguintes metricas:
  - [x] Quantidade de emails recebidos da fila do rabbit
  - [x] Quantidade de emails enviados com sucesso
  - [x] Quantidade de emails reenviados para a fila
  - [x] ~~Quantidade de emails em processamento~~(Não precisa pois é uma conta fácil de se obter, emails_recebidos - emails_enviados - emails_reenviados = emails_em_processamento)
  - [x] Tempo de envio por lote de emails
  - [x] Tamanho do payload mas mensagens recebidas
  - [x] Tamanho do email enviado
  - [ ] Tamanho do anexo enviados
  - [ ] Quantidade de anexos enviados 
  - [ ] Tamanho do cache local do minio
