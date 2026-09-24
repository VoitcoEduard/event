# Administrator authentication deployment

The production app uses Google sign-in for administrators. New Google identities start as **pending**. An owner must approve them before they can manage events. Players still use event admission and projectors remain public.

## Google configuration

1. In Google Cloud Console, create a project and configure the OAuth consent screen for the people who will administer the app. Request only `openid`, `email`, and `profile`. If the app is in Google's testing mode, add the administrators as test users.
2. Create a **Web application** OAuth client. Set the authorized redirect URI to `https://YOUR-HOST/auth/google/callback`. The scheme, hostname, and path must match `PUBLIC_BASE_URL` exactly.
3. Put the client ID and secret into Portainer environment variables or a server-managed secret file. Never commit them or place them in a Vite variable.
4. Deploy an image containing this authentication implementation. Its runtime must reach `accounts.google.com` over HTTPS and trust public CAs.

## Production Compose values

`docker-compose.prod.yml` requires `APP_IMAGE`, `PUBLIC_BASE_URL`, `GOOGLE_CLIENT_ID`, and `GOOGLE_CLIENT_SECRET`. Set `MONGO_DB_NAME` explicitly (`eventdb` is the compatibility default), `MONGO_URI`, and `TUNNEL_TOKEN` for your deployment. Keep production and staging databases and OAuth clients separate. The backend and Vue app are served from one public origin, including `/auth/*` and `/ws`.

The existing Compose network must be able to resolve and reach the separate MongoDB container named in `MONGO_URI`. Attach MongoDB to the app network or use an appropriate private URI. Do not publish MongoDB to the public internet.

The production container refuses to start with development authentication, missing Google credentials, or a non-HTTPS `PUBLIC_BASE_URL`.

## First owner and recovery

1. Sign in with the intended owner's Google account. It will show **Access pending**.
2. On the NAS, run `docker compose -f docker-compose.prod.yml exec app ./server auth list-users`. Identify that account by its displayed email and user ID.
3. Run `docker compose -f docker-compose.prod.yml exec app ./server auth promote-owner --user-id ID` with the exact ID. The operation is idempotent and records an audit event.
4. Sign out and sign in again. The owner can now approve or disable regular administrators at `/admin/users`.

Only the server command can promote an owner. Keep console access and a production database backup available for owner recovery. Disabling an administrator revokes their sessions; open privileged WebSockets close within thirty seconds or on the next privileged message.

## Development on the NAS

`docker-compose.nas-dev.yml` runs the same bundled image with `AUTH_MODE=dev`. Set `DEV_IMAGE`, `DEV_PUBLIC_BASE_URL`, `DEV_ADMIN_PASSWORD`, and `DEV_MONGO_URI`. It uses the `event_dev` database by default and binds port `8082` on the NAS. The public origin should point to its dedicated development hostname, with HTTPS if exposed outside your private network. Development passwords are only for this deployment and must not be copied to production.

Local `docker-compose.yml` uses the Vite server at `http://localhost:8081`. Set `DEV_ADMIN_PASSWORD` in a local `.env`. Vite proxies API, authentication, and WebSocket requests to the Go backend to preserve same-origin cookies.

## Release checks and rollback

Verify Google sign-in, pending approval, owner management, session logout, blocked legacy bearer tokens, admin WebSocket hosting, public player joining, and projector viewing in staging. Promote the tested image digest to production during a maintenance window. Retain a database backup and the previous authenticated release image. Roll back to that authenticated image if needed; the earlier fixed-token image would reopen the authentication bypass.
