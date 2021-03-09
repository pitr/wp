FROM alpine:latest

ADD build/linux/wp /
ADD wp-prod.crt /wp.crt
ADD wp-prod.key /wp.key

ENV LS_TOKEN="2cHQSQs1i++V2cDsuKoZlZPw8MvamvspCDwHkRwSb3zzbhi0aAP37nQTUrW2A0P2v5KE06D++MvYu8O6HmgQ+e0eJzzAIFU7mJSQAVfr"

CMD ["/wp"]