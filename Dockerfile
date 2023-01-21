FROM alpine:latest
COPY krmfnsops /usr/local/bin/config-function
CMD ["config-function"]
