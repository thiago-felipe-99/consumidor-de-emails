## Descrição
O objetivo desse projeto é ler a partir de uma fila do RabbitMQ as informações de um envio de email, se houver anexos pegar eles do Minio e enviar o e-mail formatado para um servidor SMTP

## Objetivos
- [ ] Ler destinatário, descrição, mensagem e caminho de anexos do email a partir de uma fila do RabbitMQ
- [ ] Fazer envio de email sem anexo
- [ ] Fazer envio de email com anexo
- [ ] Ler anexo do Minio
- [ ] Criar cache do Minio
