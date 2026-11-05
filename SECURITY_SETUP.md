# Configuracao segura

- O arquivo `.env` foi removido do versionamento nesta branch. Copie `.env.example` para `.env` se executar localmente, ou use variaveis de ambiente.
- A aplicacao usa valores padrao para configuracoes nao sensiveis; `REDIS_PASSWD` fica vazio por padrao para o Redis local de demonstracao. Configure autenticacao forte em ambientes reais.
- O Dockerfile nao copia mais `.env` para a imagem. O Compose injeta as configuracoes do ambiente.
- Se alguma senha do `.env` anteriormente publicado foi utilizada, altere/revogue a credencial. A remocao nesta branch nao apaga commits antigos.
- Teste a execucao com e sem `.env` e confirme conectividade com o Redis.
