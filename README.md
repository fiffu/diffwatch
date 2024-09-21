# Diffwatch

Get notified when URL + XPath changes.

## Roadmap

- [x] Create user
- [x] Create subscription
- [x] Take snapshot using URL and XPath
- [ ] Cronjob to scrape subscriptions
- [ ] Email integration to notify when subscription content changes
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
```sh
curl -v 'localhost:8080/api/users/1/subscription' -H 'Authorization: Basic YWRtaW46cGFzc3dvcmQ=' \
-F 'endpoint=https://example.com/' -F 'xpath=/html/body/div/h1'
```

Show the latest scraped content for a subscription
```sh
curl -v 'localhost:8080/api/users/:user_id/subscription/:subscription_id/latest' -H 'Authorization: Basic YWRtaW46cGFzc3dvcmQ='
```

