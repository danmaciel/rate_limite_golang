# Configuracao segura

- O arquivo `.env` foi removido do versionamento nesta branch. Copie `.env.example` para `.env` se executar localmente, ou use variaveis de ambiente.
- A aplicacao usa valores padrao para configuracoes nao sensiveis; `REDIS_PASSWD` fica vazio por padrao para o Redis local de demonstracao. Configure autenticacao forte em ambientes reais.
- O Dockerfile nao copia mais `.env` para a imagem. O Compose injeta as configuracoes do ambiente.
- Se alguma senha do `.env` anteriormente publicado foi utilizada, altere/revogue a credencial. A remocao nesta branch nao apaga commits antigos.
- O Compose publica HTTP e Redis apenas no loopback do host. Os servicos continuam acessiveis na rede Docker do exemplo; isso nao representa isolamento de producao.
- O processo trata SIGINT/SIGTERM com encerramento gracioso e timeout de 10 segundos; sinais ou encerramento forçado podem interromper requisicoes.
- O Redis do Compose usa a linha `7-alpine` por previsibilidade de compatibilidade, mas a tag nao e imutavel. Para maior reproducibilidade, fixe um digest validado em um ambiente controlado.
- Teste a execucao com e sem `.env` e confirme conectividade com o Redis.
