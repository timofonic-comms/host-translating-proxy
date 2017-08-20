# host-translating-proxy

A "host translating" reverse proxy translates request hostnames from that of the frontend (the proxy host itself) to that of the backend, and translates response hostnames from that of the backend to that of the frontend. It looks for hostnames in relevant areas of the requests and responses, such as Location and Referer headers and HTML content.

The primary goal of this proxy is to provide vanity domain names in front of a TLS-secured backend that require Host headers targeted to the backend's hostname. For example, NationBuilder provides TLS-secured servers at domains like foo.nationbuilder.com, but their application servers rely on Host headers that obey that format in order to perform backend routing. As such, you can't use a Cloudflare TLS proxy to provide a vanity domain. Instead, you need a proxy that can manage vanity hostnames and the backend hostname accordingly.

## Installation

```
go get github.com/jeffomatic/host-translating-proxy
```

## Running

```
BACKEND_URL=https://backend.example.com PORT=3000 host-translating-proxy
```

## Development

Ports aren't handled very gracefully (see [TODO](#todo) below), so if you're developming locally, you'll want to use ngrok or something similar so you can host a local proxy at the standard HTTP ports.

## TODO

- handle ports gracefully (currently assumes 80 for HTTP and 443 for HTTPS).
- add Let's Encrypt and TLS support to proxy
- cache backend responses
- cache inbound requests for web assets (images, styles, CSS, and favicons)
