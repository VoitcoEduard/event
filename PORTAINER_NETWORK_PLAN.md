# One NAS, multiple projects, one Cloudflare Tunnel

Use standalone Docker Compose stacks on the same NAS Docker environment.
Create one external bridge network named `edge`; keep each project's database
on its own private network. No extra reverse proxy is required for hostname
routing: the tunnel maps each public hostname directly to a web container.

```text
Internet HTTPS
    |
Cloudflare Tunnel (infrastructure stack)
    |
edge (shared external Docker bridge)
    +-- quiz-prod-web:8080 -- quiz-prod private network -- PostgreSQL
    +-- another-project-web:PORT -- its private network -- its database
```

All web containers on edge can reach one another. This is appropriate for your
own trusted projects; it is not tenant isolation. Databases join only their
project network. Use unique edge aliases across all stacks and route to those
aliases rather than generic service names such as `app` or `web`. Docker assigns
container IPs automatically, so do not configure routes with fixed container IPs.

## Apply to the NAS shown in the screenshots

1. Keep Portainer and Tailscale available for NAS administration. Export the old
   event stack configuration and preserve its volumes before retiring it.
2. Portainer > Networks > Add network: name `edge`, driver `bridge`, automatic
   subnet allocation. Create this once in the same environment as the stacks.
3. Deploy `docker-compose.infrastructure.yml` as stack `infrastructure`, using
   the Web editor and setting `TUNNEL_TOKEN`. Choose a tested image tag or digest
   through `CLOUDFLARED_IMAGE` if you want controlled connector upgrades.
4. Deploy this project as `quiz-prod` from Git using
   `docker-compose.asustor.yml`. Follow `ASUSTOR_FRESH_INSTALL.md` for folders,
   credentials, release images, Google login and first-owner activation. This
   file includes a repository-relative database initialization script, so use
   Git deployment rather than pasting only its YAML into the Web editor.
5. In the tunnel's published application routes, map your quiz hostname to HTTP
   service `quiz-prod-web:8080`. Public HTTPS terminates at Cloudflare. No app or
   PostgreSQL host port is required.
6. Confirm login, player/projector live updates, uploads and persistence after a
   restart. Only then retire the previous event containers. Do not delete their
   data as part of the cutover.

If replacing an existing tunnel connector, inventory its routes first. Every
active connector replica must reach the same configured origins. Replace the
old connector during cutover; leaving a connector with different network access
can cause intermittent origin failures. The screenshots do not show whether a
connector exists on the second container page or outside Docker.

## Existing containers and networks

| Screenshot item | Action |
| --- | --- |
| PortainerCE, TailScale | Keep running; use private administration access. |
| Immich containers / `imchdocker_immich.net` | Keep their existing private networking and data. Join only the HTTP server to edge if publishing it through this tunnel. |
| Nextcloud (restarting) | Diagnose logs separately before routing traffic to it. |
| `go_server_app`, `go_server_db`, `event_mongodb` | Identify stack ownership and dependencies before stopping anything; the new quiz deployment uses fresh PostgreSQL storage and does not need MongoDB. |
| `go-game_game_net` | Keep while any existing stack uses it. The screenshot shows multiple stacks sharing addresses in this subnet. |
| `ptnrce_default` | Keep for Portainer. |
| `bridge`, `host`, `none` | Leave Docker's system networks in place. |

Do not remove networks simply to make the list shorter. Remove an old custom
network only after its owning stack is retired and it has no attached containers.
Do not run a global volume prune or reset Docker for this fresh project install.

## Add another project later

Adapt `deploy/compose.shared-edge.example.yml` into that project's own Compose
file. Set its image and unique alias, retain its required volumes/environment,
and attach only its HTTP service to edge. Attach its database only to private.
The template's private network has no external connectivity; the web service
also has edge for outbound connectivity.

Add another published application route on the **same tunnel**, for example
`inventory.YOUR-DOMAIN` to `http://inventory-prod-web:3000` (use the actual
container listening port). No additional connector, host port, router forwarding,
static IP or new shared network is needed. Persist the edge attachment in each
stack's Compose file so redeployments retain it.

For an existing project, the relevant additions are:

```yaml
services:
  your-web-service:
    networks:
      default: {}
      edge:
        aliases: [inventory-prod-web]
networks:
  default: {}
  edge:
    external: true
    name: edge
```

Merge these into its existing networks rather than replacing custom network
definitions. Databases must not join edge. Use application-specific trusted
proxy/public URL settings where required. Keep NAS/Portainer administration
private or protect any deliberately published administration route with access
control.

## Verification

- In Portainer, inspect edge: cloudflared and intended web containers should be
  attached; database containers should be absent.
- Confirm the tunnel connector is healthy in Cloudflare and inspect its logs in
  Portainer if an origin cannot be reached.
- Open each configured hostname and test its real workflows. A 502 usually
  requires checking the alias, container port, network membership and app logs.
- Redeploy one application and confirm other applications remain reachable.
- Recreate the quiz container and confirm its media and database persist.

The NAS stack is `docker-compose.asustor.yml`. The older generic
`docker-compose.prod.yml` includes a project-local connector/network; do not
deploy it alongside this shared infrastructure setup.

References:
- https://docs.docker.com/compose/how-tos/networking/
- https://developers.cloudflare.com/tunnel/concepts/routing/
