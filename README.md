# rate_limite_golang
Rate limite escrito em golang que utiliza IP e TOKEN para controle

## O que preciso para rodar?

Para rodar o projeto é necessário o Docker e Go instalados e configurados previamente.

## Como usar?

Entre na raiz do projeto e execute o seguinte comando subir o Redis e a aplicação via Docker:

```
docker-compose up
```

Se tudo estiver der certo, a aplicação responderá em http://localhost:8080

## Como Funciona?

Quando a aplicação estiver rodando em http://localhost:8080, ela poderá receber requisições em / e irá considerar para um mesmo IP ou token uma quantidade predefinida de requisições, se ultrapassar a quantidade de requisições definida, a aplicação ira bloquear as requisições por IP ou token por um tempo determinado e previamente configurado, retornando com o código 429 e a mensagen "you have reached the maximum number of requests or actions allowed within a certain time frame"
As informações utilizadas para a validação podem ser configuradas através do arquivo .env na raiz da aplicação, sendo elas:

```
WEB_SERVER_PORT                 -> porta da aplicacao, sendo o padrao 8080
MAX_REQUISITIONS_BY_IP          -> a quantidade de requisicoes aceita por ip, sendo o padrao 5
BLACK_LIST_MINUTES_BY_IP        -> o tempo em minutos do bloqueio por ip, sendo o padrao 2
BLACK_LIST_MINUTES_BY_TOKEN     -> o tempo em minutos do bloqueio por token, sendo o padrao 2
REDIS_ADDR                      -> porta do Redis, sendo o padrao 8080
REDIS_PASSWD                    -> senha do banco Redis(texto), sendo o padrao vazio
REDIS_DB                        -> banco do redis que sera utilizado, sendo o padrao 0
```

Também é possível configurar a quantidade de requisições por IP, tempo de bloqueio por IP, tempo de bloqueio por token por variáveis de ambiente, sendo elas:

```
MAX_REQUISITIONS_BY_IP          -> a quantidade de requisicoes aceita por ip
BLACK_LIST_MINUTES_BY_IP        -> o tempo em minutos do bloqueio por ip
BLACK_LIST_MINUTES_BY_TOKEN     -> o tempo em minutos do bloqueio por token
```

Se estes parâmetros não estiverem nas variáveis de ambiente, a aplicação irá carregar por padrão o arquivo .env

As informações utilizadas para controle do rate limit estão sendo gravadas e lidas do Redis.

No arquivo redis_strategy_test.go disponível no diretorio tests, existem alguns testes que demonstram o funcionamento da aplicação