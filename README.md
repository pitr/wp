# Wikipedia mirror for Gemini protocol

## Requirements

- Go v1.14+

## Development

```bash
openssl req -x509 -out wp.crt -keyout wp.key -newkey rsa:2048 -nodes -sha256 -subj '/CN=localhost' -extensions EXT -config <( printf "[dn]\nCN=localhost\n[req]\ndistinguished_name = dn\n[EXT]\nsubjectAltName=DNS:localhost\nkeyUsage=digitalSignature\nextendedKeyUsage=serverAuth")
```
