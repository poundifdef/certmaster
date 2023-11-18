# Certmaster

Certmaster automates the process of renewing and installing SSL certificates.

1. Creates an SSL cert from Let's Encrypt
2. Validates certs using DNS
3. Uploads or installs SSL certs to multiple destinations

## Supported DNS

Certmaster uses the excellent [go-acme/lego](https://github.com/go-acme/lego) repository
to generate certificates and automate DNS validation. They support 100+ providers, which are 
listed [here](https://go-acme.github.io/lego/dns/).

## Supported Destinations

1. Email
2. SFTP
3. Hetzner Load Balancer

## Config

Start with the example [config.json](config.json) and modify it.

- To configure DNS providers, create JSON of the form:

    ``` json
    {
        "provider": "route53",
        "AWS_ACCESS_KEY_ID": "ACCESS_KEY_ID",
        "AWS_SECRET_ACCESS_KEY": "SECRET_KEY"
    }
    ```

    Here, `provider` is the provider name from `go-acme/lego`'s documentation. The rest of the
    fields are configs specific to your DNS provider.

- Similarly, you configure destinations with all details required to upload. 

## Usage

To update the certificate, just run:

```
$ ./certmaster create --config config.json
```

### AWS Lambda

The Docker file is to use with AWS Lambda. When you invoke the function, 
you send the same JSON payload as the normal config.
