# MiniMirror
Simple and lightweight go lang microservice to create mirror of a website to bypass censorship.

Now only get requests are supported, we plan to add other types, but can't guarantee anything :) 
## Dev

`TARGET_DOMAIN=https://example.com SECONDARY_DOMAINS=https://s3.example.com go run MiniMirror`

## Production

`go build`

`./MiniMirror`


## Production 

`go build`

`./MiniMirror`

## Production in Docker
`docker run -it -p 8080:8080 -e TARGET_DOMAIN=https://example.com -e SECONDARY_DOMAINS=https://s3.example.com  $(docker build -q .)`