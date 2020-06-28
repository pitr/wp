FROM alpine:latest

ADD build/linux/wp /
ADD wp.crt /
ADD wp.key /

CMD ["/wp"]