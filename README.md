## Descrição
O objetivo desse projeto é ler a partir de uma fila do RabbitMQ as informações de um envio de email, se houver anexos pegar eles do Minio e enviar o e-mail formatado para um servidor SMTP

## Objetivos
- [ ] Ler destinatário, descrição, mensagem e caminho de anexos do email a partir de uma fila do RabbitMQ
- [ ] Fazer envio de email sem anexo
- [ ] Fazer envio de email com anexo
- [ ] Ler anexo do Minio
- [ ] Criar cache do Minio

## Metricas
- [ ] Expor as metricas do servidor na porta `8001`
- [ ] Pegar as seguintes metricas:
  - [ ] Quantidade de emails recebidos
  - [ ] Quantidade de emails enviados
  - [ ] Quantidade de emails reenviados para a fila
  - [ ] Quantidade de emails em processamento
  - [ ] Tamanho do payload mas mensagens recebidas
  - [ ] Tamanho do email enviado
  - [ ] Tamanho do anexo enviados
  - [ ] Quantidade de anexos enviados 
  - [ ] Tamanho do cache local do minio
