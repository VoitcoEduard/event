# Automatic deployment to an Asustor NAS

The NAS only pulls and runs prebuilt containers. GitHub Actions builds the Go
backend and Vue frontend for `linux/amd64`, publishes the image to GitHub
Container Registry (GHCR), and then advances a generated deployment branch.
Portainer polls that branch and updates the corresponding stack.

| Source branch | Image tag | Generated branch | Portainer stack |
| --- | --- | --- | --- |
| `develop` | `ghcr.io/OWNER/REPOSITORY:dev` | `deploy-dev` | NAS development |
| `main` | `ghcr.io/OWNER/REPOSITORY:prod` | `deploy-prod` | Production |

The publish workflow runs only after the existing CI workflow succeeds. It also
publishes an immutable `sha-COMMIT` tag for rollback. The generated deployment
branches are replaced by automation and must not be edited manually.

## 1. GitHub setup

Push the repository to GitHub. In **Settings > Actions > General > Workflow
permissions**, select **Read and write permissions** so the workflow can publish
the package and advance `deploy-dev` and `deploy-prod`.

Create the `develop` branch from `main`. Day-to-day changes go through
`develop`; merging the tested commit into `main` promotes it to production.

The first successful run creates the GHCR package and the matching generated
deployment branch. The image name is the lowercase GitHub repository name. For
example, `https://github.com/acme/quiz` produces `ghcr.io/acme/quiz:dev` and
`ghcr.io/acme/quiz:prod`.

The simplest NAS setup is to make the GHCR package public. If it remains
private, add `ghcr.io` to Portainer as a custom registry and authenticate with a
GitHub token that can read packages. Do not put that token in the repository.

## 2. Development stack in Portainer

After the first successful `develop` workflow run:

1. Open **Stacks > Add stack > Git repository**.
2. Name it `quiz-dev` and enter the GitHub repository URL.
3. For a private repository, enable authentication and provide a read-only
   repository token.
4. Set **Repository reference** to `refs/heads/deploy-dev`.
5. Set **Compose path** to `docker-compose.nas-dev.yml`.
6. Enable **GitOps updates**, choose **Polling**, and use a five-minute interval.
7. Enable **Re-pull image**. Leave **Force redeployment** disabled.
8. Add the environment variables below and deploy the stack.

```dotenv
DEV_IMAGE=ghcr.io/OWNER/REPOSITORY:dev
DEV_PUBLIC_BASE_URL=https://dev.example.com
DEV_ADMIN_PASSWORD=use-a-long-development-password
DEV_POSTGRES_PASSWORD=development-admin-password
DEV_APP_DB_PASSWORD=development-app-password
DEV_DATABASE_URL=postgresql://quiz_app:development-app-password@postgres:5432/quiz_dev?sslmode=disable
```

The development app binds NAS port `8082`. Its PostgreSQL and media folders are separate from production. Create the media
folder with ownership UID/GID 10001 and use different development credentials.

## 3. Production stack in Portainer

After the first successful `main` workflow run, repeat the process with:

- stack name: `quiz-prod`
- repository reference: `refs/heads/deploy-prod`
- Compose path: `docker-compose.prod.yml`
- polling enabled, with **Re-pull image** enabled and **Force redeployment**
  disabled

Set these variables in Portainer:

```dotenv
APP_IMAGE=ghcr.io/OWNER/REPOSITORY:prod
PUBLIC_BASE_URL=https://events.example.com
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
POSTGRES_PASSWORD=production-admin-password
APP_DB_PASSWORD=production-app-password
DATABASE_URL=postgresql://quiz_app:production-app-password@postgres:5432/quiz_prod?sslmode=disable
TUNNEL_TOKEN=your-cloudflare-tunnel-token
```

Register `https://events.example.com/auth/google/callback` in the Google OAuth
client. PostgreSQL runs on a private network without a published port. The application
password must match the password in DATABASE_URL. For the shared edge tunnel
setup, follow `ASUSTOR_FRESH_INSTALL.md` and use `docker-compose.asustor.yml`.

## Normal release flow

1. Push or merge changes into `develop`.
2. CI passes, the `dev` image is published, and Portainer updates `quiz-dev` on
   its next poll.
3. Test the development deployment.
4. Merge the tested commit from `develop` into `main`.
5. CI passes again, the `prod` image is published, and Portainer updates
   `quiz-prod` on its next poll.

Portainer's webhook-triggered deployment is a Business Edition feature. Polling
keeps Portainer private and works without exposing its API to GitHub Actions.
Because the deployment branch advances only after the image upload completes,
polling does not race the container build.

## Rollback

Every published image has an immutable `sha-COMMIT` tag. In Portainer, change
`APP_IMAGE` or `DEV_IMAGE` temporarily to the last known-good tag, for example:

```dotenv
APP_IMAGE=ghcr.io/OWNER/REPOSITORY:sha-0123456789abcdef
```

Then use **Pull and redeploy**. Keep the previous production image and a PostgreSQL and media
backup until the new release has been verified.
