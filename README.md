# Diffwatch

Get notified when URL + XPath changes.

## Roadmap

- [x] Create user
- [x] Create subscription
- [x] Take snapshot using URL and XPath
- [x] Cronjob to scrape subscriptions
- [x] Email integration to notify when subscription content changes
- [ ] ~~Switch from sqlite to Postgres~~ (won't do: this remains a locally-hosted project for now, since many sites block VPS traffic)
- [x] Add `name` attribute for subscriptions (or detect by html title tag)
- [ ] Security checks, e.g. max retrievable content size
- [ ] Other notification integrations, e.g. webhooks
- [ ] Support JSON endpoints (workaround: support via [j2x-proxy](https://github.com/fiffu/j2x-proxy))

## Development

Start server
```sh
cp .env.development .env
source .env && go run main .
```

Create user and verify email address using the nonce.
The Basic Auth is configured in the `.env` file.
```sh
curl -v 'localhost:8080/api/users' -H 'Authorization: Basic YWRtaW46cGFzc3dvcmQ=' \
-F 'email=somebody@gmail.com' -F 'password=12345'

curl -v 'localhost:8080/verify/11111111-1111-1111-1111-111111111111'
```

Create a subscription. This only succeeds when:
1. The endpoint can be fetched correctly by plain HTTP GET
2. The XPath successfully returns some non-empty text content

> **XPath tips**
>
> Usually it's simpler to use Inspect Element on your browser on the element of interest, then use "Copy > XPath".
>
> For help with XPath syntax: https://www.w3schools.com/xml/xpath_syntax.asp

```sh
curl -v 'localhost:8080/api/users/1/subscription' -H 'Authorization: Basic YWRtaW46cGFzc3dvcmQ=' \
-F 'endpoint=https://example.com/' -F 'xpath=/html/body/div/h1'
```

Show the latest scraped content for a subscription
```sh
curl -v 'localhost:8080/api/users/:user_id/subscription/:subscription_id/latest' -H 'Authorization: Basic YWRtaW46cGFzc3dvcmQ='
```

