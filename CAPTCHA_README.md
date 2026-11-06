# CAPTCHA — Cloudflare Turnstile (AlephK / Service Hub)

Added on 2026-10-25 (control point **CC 2**). No database changes are needed.

## 1. What is protected

| Screen | Form | Checked by |
|---|---|---|
| Store (showcase) | Cart: guest order and signed-in order | shao |
| Store (showcase) | Booking window | shao |
| p3 | "Your plan" → Request a plan | shao |
| p3 | Support request form | usersAppAK |

The browser sends the token in the header `X-Captcha-Token`. Each token works **only once**, so the box gives a new one after every attempt.

## 2. The two keys (Cloudflare dashboard → Turnstile → your widget → Settings)

| Key | Public? | Where it goes |
|---|---|---|
| **Site key** | Yes | Frontends (showcase and p3), only for the production build |
| **Secret key** | **No, never share it** | Backends (shao and usersAppAK), in `.env` |

Never put the Secret key in the frontends, in git, or in a chat.

## 3. On your computer / LAN (testing)

Nothing to configure in the frontends: on `localhost`, `127.0.0.1`, `192.168.x.x`, `10.x.x.x` and `172.16–31.x.x` the apps
automatically use Cloudflare's **test site key** (`1x00000000000000000000AA`, always passes).

In the backends choose ONE option in `shao/.env` and `usersAppAK/.env`:

```env
# Option A — CAPTCHA off (simplest):
TURNSTILE_SECRET_KEY=

# Option B — test the whole flow (test secret, always passes):
TURNSTILE_SECRET_KEY=1x0000000000000000000000000000000AA
```

⚠️ Do **not** use your real Secret key on your computer: it rejects the tokens of the test site key.

## 4. In production

1. **Cloudflare:** in the widget settings, the **Hostnames** must include your real addresses, for example
   `store.yourdomain.com` and `app.yourdomain.com`.
2. **shao** — `shao/.env` (next to the shao binary):
   ```env
   TURNSTILE_SECRET_KEY=<your Secret key>
   ```
3. **usersAppAK** — `usersAppAK/.env`:
   ```env
   TURNSTILE_SECRET_KEY=<your Secret key>
   ```
4. **showcase** and **p3** — `.env.production` in each project, then build again:
   ```env
   REACT_APP_TURNSTILE_SITE_KEY=<your Site key>
   ```
   ```bash
   npm run build
   ```
   (React copies the variables into the build; changing `.env.production` needs a new build.)
5. Restart shao and usersAppAK.

If you use systemd, Docker or Cloud Run instead of `.env`, set the same variables there.

## 5. Optional settings (backends)

| Variable | Default | Meaning |
|---|---|---|
| `CAPTCHA_PROVIDER` | `turnstile` | `none` turns the CAPTCHA off |
| `CAPTCHA_FAIL_OPEN` | empty (refuse) | `1` = if Cloudflare cannot be reached, let the request pass |
| `TURNSTILE_ALLOWED_HOSTNAMES` | empty (any) | e.g. `store.yourdomain.com,app.yourdomain.com`: accept only tokens solved there |

## 6. Quick test

1. Start shao, usersAppAK, the store and p3.
2. Store → add a product → cart: the Turnstile box appears above the send button; send the order → it works.
3. Booking window and p3 support / plan request forms: same.
4. With Option B (test secret), set `TURNSTILE_SECRET_KEY=2x0000000000000000000000000000000AA` (always fails),
   restart, send an order → you must see "The security check failed…". Put Option A or B back after the test.

## 7. Messages and codes

| Code | HTTP | When |
|---|---|---|
| `CAPTCHA_REQUIRED` | 400 | CAPTCHA on and no token sent |
| `CAPTCHA_FAILED` | 400 | Token invalid, expired, used twice or wrong hostname |
| `CAPTCHA_UNAVAILABLE` | 503 | Cloudflare could not be reached (fail closed) |

Every failure is tracked (`lt`): `PublicCaptchaFailed`, `CreatePlanRequestCaptchaFailed`, `SupportRequestCaptchaFailed`.

## 8. Problems

| Problem | Cause / fix |
|---|---|
| Production: "Please complete the security check" but no box | `REACT_APP_TURNSTILE_SITE_KEY` missing at build time → add it and build again |
| Every request fails with `CAPTCHA_FAILED` | Site key and Secret key from different widgets, or test key ↔ real secret mixed, or hostname not added in Cloudflare |
| Box shows an error (e.g. 110200) | The address is not in the widget hostnames |
| `CAPTCHA_UNAVAILABLE` | Server cannot reach `challenges.cloudflare.com` (firewall / internet) |

## 9. Files

- shao: `bussunessLedger/blcaptcha/bCaptchaVerify.go` (the replaceable piece), `HandlerLedgers/hPublicShowCase/publicFormLimits.go`,
  `HPublicWebOrderRequest.go`, `HPublicAppointmentRequest.go`, `HandlerLedgers/planinshao/HCreatePlanRequest.go`, `router/router.go`
- usersAppAK: `brules/bCaptcha/bCaptchaVerify.go`, `handlers/HsupportRequest/HCreateSupportRequest.go`, `routers/router.go`
- showcase: `src/App/shared/Hooks/useCaptcha.js`, `CompanyShowcase.jsx`, `WebOrderRequestService.js`, `AppointmentRequestService.js`
- p3: `src/Shared/Hooks/useCaptcha.js`, `SupportRequestForm.jsx`, `SupportRequestService.js`, `PlansModal.jsx`, `CompanyPlanService.js`

To change the provider later (hCaptcha, reCAPTCHA): only `bCaptchaVerify.go` (backends) and `useCaptcha.js` (frontends) change.

## 10. Undo (back to CC 2)

Follow `.ai_backups/CC2_before_captcha/README.txt` in each project.
