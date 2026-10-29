# Fresh Asustor / Portainer deployment

This installs the current bundled Go + Vue application with an empty database,
Google administrator login, a shared Cloudflare tunnel, and GitHub-driven releases.
It does not reuse the old GitLab image, old admin password, old containers, or old
database folder. Keep Portainer and Tailscale running throughout.

See [PORTAINER_NETWORK_PLAN.md](PORTAINER_NETWORK_PLAN.md) for the NAS-wide
network layout, handling the existing containers, and adding other projects to
the same tunnel. Use `docker-compose.asustor.yml` with the independent
infrastructure stack for this installation.

## 1. Database and media storage

PostgreSQL 18 runs locally on the NAS, on a private Docker network. This is a
fresh installation: no old database or uploaded files are imported. Production
and development use different database and media folders and credentials.

## 2. Prepare the current code on GitHub

Having CI locally does not mean GitHub has these files. Review, commit, and push
the intended current project, including:

- `.github/workflows/ci.yml` and `.github/workflows/publish.yml`
- `Dockerfile`, backend, frontend, and required dependency files
- `deploy-control/` (the publishing workflow currently builds this image on main,
  even though running the controller is optional)
- `docker-compose.asustor.yml`, `deploy/postgres/`, and `scripts/`
- `docker-compose.infrastructure.yml` and `PORTAINER_NETWORK_PLAN.md`

Never commit `.env` or credentials. This checkout currently has no Git remote
configured; add your actual GitHub repository as origin if that remains the case.

Use `main` as the GitHub default branch and ensure publish.yml exists there:
GitHub workflow_run triggers require the workflow on the default branch.

Push the current intended production commit to main. In Actions, wait for:

1. **CI** to pass.
2. **Publish and stage deployment** to pass.

The workflow uses GITHUB_TOKEN with contents:write and packages:write; repository
or organization policy must allow those permissions. No Docker Hub/GitLab
password is needed for publishing to GHCR through this workflow.

Confirm these outputs exist before continuing:

- Image `ghcr.io/YOUR-OWNER/YOUR-REPOSITORY:prod` (lowercase).
- Immutable image `ghcr.io/YOUR-OWNER/YOUR-REPOSITORY:sha-FULL-COMMIT-SHA`.
- Git branch `deploy-prod`, containing `docker-compose.asustor.yml`.

For a private image, add a custom registry in Portainer: URL ghcr.io, your GitHub
username, and a personal access token (classic) with read:packages and access to
the package. Private Git repository access is a separate credential setting.

References: https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry
and https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows

## 3. Retire only the old event deployment

In Portainer, stop the old event app/stack once ready for downtime. Export its
configuration first if wanted. Keep the existing Portainer, Tailscale, Immich,
Nextcloud, and unrelated game containers. There is no reason to reset Docker.

Create empty NAS folders `/volume1/Docker/quiz-prod/postgresql` and
`/volume1/Docker/quiz-prod/media`. The application runs as UID and GID 10001;
assign ownership of the media folder to 10001:10001 (for example, run
`sudo chown 10001:10001 /volume1/Docker/quiz-prod/media` on the NAS). PostgreSQL manages its own
folder permissions. For development use separate `quiz-dev/postgresql` and
`quiz-dev/media` folders with the same media ownership.

Generate separate long random alphanumeric administrator and application database
passwords, so the latter can be used in a URL without escaping. Store both in a
password manager and Portainer, never in Git.

## 4. Create the shared edge network

In Portainer select the NAS Docker environment, then Networks > Add network:

- Name: `edge`
- Driver: `bridge`
- Leave subnet allocation automatic.

Create it once. The two new Compose files reference it as an external network.

## 5. Deploy the independent tunnel

In Cloudflare create one remotely managed Cloudflared tunnel for the NAS, or
reuse an existing tunnel after checking its current routes. Copy its connector
token. No router port forwarding is required.

Portainer > Stacks > Add stack:

- Name: `infrastructure`
- Method: Web editor
- Paste `docker-compose.infrastructure.yml`.
- Add environment variable TUNNEL_TOKEN with the connector token.
- Deploy and confirm Cloudflare reports the connector connected/healthy.

If reusing an existing tunnel, any old connector replicas must be able to reach
the same origins. Stop/replace the old connector when cutting over, or requests
can be routed to a connector that cannot reach the new app. Preserve its other
routes while migrating them.

In the tunnel's published applications, add/update:

- Hostname: `quiz.YOUR-DOMAIN`
- Service type: HTTP
- URL: `quiz-prod-web:8080`

Until the app starts this route can show an origin error. Use the container
address, not localhost, and use HTTP internally; Cloudflare supplies public HTTPS.
Do not put an interactive Cloudflare Access login in front of the entire app if
players/projectors must access it publicly.

## 6. Create Google login credentials

In Google Cloud Console, configure OAuth branding/audience and create an OAuth
client of type Web application. Use your intended real HTTPS hostname:

- Authorized redirect URI: `https://quiz.YOUR-DOMAIN/auth/google/callback`
- If entering an authorized JavaScript origin: `https://quiz.YOUR-DOMAIN`
- If the OAuth app is in testing mode, add your account as a test user.

Request only openid, email, and profile. Save the client ID and client secret.
The redirect origin must exactly match PUBLIC_BASE_URL below.

## 7. Deploy production from Git

Portainer > Stacks > Add stack > Git repository:

| Setting | Value |
| --- | --- |
| Stack name | quiz-prod |
| Repository URL | Your actual GitHub repository HTTPS URL |
| Repository reference | refs/heads/deploy-prod |
| Compose path | docker-compose.asustor.yml |
| GitOps updates | Enabled |
| Mechanism | Polling |
| Fetch interval | 5 minutes |
| Re-pull image | Enabled |
| Force redeployment | Disabled |

Add repository authentication if the Git repository is private. Ensure the
registry credential is available/selected for GHCR if the image is private.

Add these environment variables in Portainer; do not leave placeholders:

```dotenv
APP_IMAGE=ghcr.io/YOUR-OWNER/YOUR-REPOSITORY:prod
PUBLIC_BASE_URL=https://quiz.YOUR-DOMAIN
GOOGLE_CLIENT_ID=YOUR-CLIENT-ID
GOOGLE_CLIENT_SECRET=YOUR-CLIENT-SECRET
POSTGRES_PASSWORD=YOUR-NEW-DATABASE-ADMIN-PASSWORD
APP_DB_PASSWORD=YOUR-NEW-APPLICATION-DATABASE-PASSWORD
DATABASE_URL=postgresql://quiz_app:YOUR-NEW-APPLICATION-DATABASE-PASSWORD@postgres:5432/quiz_prod?sslmode=disable
```

The password in `DATABASE_URL` must match `APP_DB_PASSWORD`. PostgreSQL creates a
non-superuser application account on first initialization; changing these values
later does not rotate an existing account. The database has no published host
port. The app also joins `edge` to reach the shared tunnel. It applies pending
migrations before serving requests.

Reference: https://docs.portainer.io/user/docker/stacks/add

## 8. Activate your first owner

Open `https://quiz.YOUR-DOMAIN`, sign in with Google, and expect Access pending.

In Portainer > Containers, select the quiz-prod APP container > Console. Choose
`/bin/sh`, connect, then run:

```sh
./server auth list-users
```

Identify your Google email and copy its user ID. Then run:

```sh
./server auth promote-owner --user-id YOUR-EXACT-USER-ID
```

Sign out and sign in again. The owner can approve other administrators at
`/admin/users`. ADMIN_PASSWORD from the old deployment is not used in production.

## 9. Verify and update

Verify Google login, creation of an event, a player joining from another device,
projector view, image upload, and live updates. Restart the stack and confirm
the event survives. Check `/health/live` and `/health/ready`; the latter checks
the database, schema, and media folder. The database should have no published port.

For future production updates: push/merge into main, wait for CI and publishing,
then let Portainer detect the changed deploy-prod branch and pull the new image.
Do not edit deploy-prod manually: automation replaces it. Do not configure the
optional deployment-controller secrets for this polling-only setup.

For rollback, temporarily disable GitOps updates, set APP_IMAGE to the previous
known-good sha-FULL-COMMIT-SHA image, and pull/redeploy. Database/schema changes
can also require restoring a backup. Resume updates only when ready to advance.

Before upgrades, run the backup script on the NAS with the same Compose file,
project name, and environment as the running stack:

```sh
COMPOSE_FILE=docker-compose.asustor.yml COMPOSE_PROJECT_NAME=quiz-prod scripts/backup.sh /safe/backups/quiz-2026-01-01
```

The script stops application writes, captures PostgreSQL and media, records the
image and schema version, and restarts the app. Back up the Compose environment
and secrets separately. To test recovery, configure a **separate** stack with
empty database and media volumes and the exact same image, then run:

```sh
COMPOSE_FILE=/path/to/separate-compose.yml COMPOSE_PROJECT_NAME=quiz-restore scripts/restore-empty.sh /safe/backups/quiz-2026-01-01
```

The restore script refuses nonempty volumes and mismatched images. Verify data,
media, readiness, and container recreation before release. There are no automatic
schema downgrades; restore the matching backup with its matching image.

## Troubleshooting

- Image pull denied: check package visibility and Portainer GHCR credentials.
- Missing deploy-prod branch: check the publishing workflow exists on main and
  completed successfully after a push; a CI run for a pull request does not publish.
- Missing edge network: create edge in the same NAS Docker environment.
- PostgreSQL or media permission denied: check folder ownership and Docker
  access; media must be writable by UID/GID 10001.
- Database authentication failed: verify the initialized credentials; changing
  initialization variables does not reset an existing database account.
- Cloudflare 502: check app logs, tunnel network membership, and the exact origin
  quiz-prod-web:8080. localhost inside cloudflared refers to cloudflared itself.
- Google redirect mismatch: compare the redirect URI, public hostname, and
  PUBLIC_BASE_URL exactly; use HTTPS and no trailing slash in PUBLIC_BASE_URL.
- Access pending: perform first-owner promotion above and sign in again.
