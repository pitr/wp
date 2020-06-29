FROM alpine:latest

ADD build/linux/wp /
ADD wp-prod.crt /wp.crt
ADD wp-prod.key /wp.key

CMD ["/wp"]