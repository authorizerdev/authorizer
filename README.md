

# Authorizer

**Authorizer** is an open-source authentication and authorization solution for your applications. Bring your database and have complete control over the user information. You can self-host authorizer instances and connect to any database (Currently supports 11+ databases including [Postgres](https://www.postgresql.org/), [MySQL](https://www.mysql.com/), [SQLite](https://www.sqlite.org/index.html), [SQLServer](https://www.microsoft.com/en-us/sql-server/), [YugaByte](https://www.yugabyte.com/),  [MariaDB](https://mariadb.org/), [PlanetScale](https://planetscale.com/), [CassandraDB](https://cassandra.apache.org/_/index.html), [ScyllaDB](https://www.scylladb.com/), [MongoDB](https://mongodb.com/), [ArangoDB](https://www.arangodb.com/)).

For more information check:

- [OAuth 2.0 / OIDC Endpoint Reference](docs/oauth2-oidc-endpoints.md) – standards-compliant endpoint docs with examples
- [Migration Guide (v1 → v2)](MIGRATION.md) – configuration changes, CLI flags, deprecated APIs
- [Docs (v1 – legacy)](http://docs.authorizer.dev/)
- [Discord Community](https://discord.gg/Zv2D5h6kkK)
- [Contributing Guide](.github/CONTRIBUTING.md)

> **v2 note:** Authorizer v2 uses **CLI arguments** for all configuration. The server does **not** read from `.env` or OS env. Pass config when starting the binary (e.g. `./authorizer --client-id=... --client-secret=...`). See [MIGRATION.md](MIGRATION.md).

# Introduction



#### We offer the following functionality

- ✅ Sign-in / Sign-up with email ID and password
- ✅ Secure session management
- ✅ Email verification
- ✅ OAuth2 and OpenID compatible APIs
- ✅ APIs to update profile securely
- ✅ Forgot password flow using email
- ✅ Social logins (Google, Github, Facebook, LinkedIn, Apple more coming soon)
- ✅ Role-based access management
- ✅ Password-less login with magic link login
- ✅ Multi factor authentication
- ✅ Email templating
- ✅ Webhooks

## Roadmap

- [VueJS SDK](https://github.com/authorizerdev/authorizer-vue)
- [Svelte SDK](https://github.com/authorizerdev/authorizer-svelte)
- [Golang SDK](https://github.com/authorizerdev/authorizer-go)
- React Native SDK
- Flutter SDK
- Android Native SDK
- iOS native SDK
- Python SDK
- PHP SDK
- WordPress plugin
- Kubernetes Helm Chart
- [Local Stack](https://github.com/localstack/localstack)
- AMI
- Digital Ocean Droplet
- Azure
- Render
- Edge Deployment using Fly.io
- Password-less login with mobile number and OTP SMS

# Getting Started

## Step 1: Get Authorizer Instance

### Deploy Production Ready Instance

Deploy production ready Authorizer instance using one click deployment options available below


| **Infra provider** |                                                                                            **One-click link**                                                                                            |               **Additional information**               |
| :----------------: | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------: | :----------------------------------------------------: |
|    Railway.app     |                    [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/nwXp1C?referralCode=FEF4uT&utm_medium=integration&utm_source=template&utm_campaign=generic)                     | [docs](https://docs.authorizer.dev/deployment/railway) |
|       Heroku       | <a href="https://heroku.com/deploy?template=https://github.com/authorizerdev/authorizer-heroku"><img src="https://www.herokucdn.com/deploy/button.svg" alt="Deploy to Heroku" style="height: 44px;"></a> | [docs](https://docs.authorizer.dev/deployment/heroku)  |
|       Render       |                     [![Deploy to Render](https://render.com/images/deploy-to-render-button.svg)](https://render.com/deploy?repo=https://github.com/authorizerdev/authorizer-render)                      | [docs](https://docs.authorizer.dev/deployment/render)  |
|       Koyeb       | <a target="_blank" href="https://app.koyeb.com/deploy?name=authorizer&type=docker&image=docker.io/lakhansamani/authorizer&env[PORT]=8000&env[DATABASE_TYPE]=postgres&env[DATABASE_URL]=CHANGE_ME&ports=8000;http;/"><img alt="Deploy to Koyeb" src="https://www.koyeb.com/static/images/deploy/button.svg" /></a> | [docs](https://docs.authorizer.dev/deployment/koyeb)  |
|     RepoCloud     | <a href="https://repocloud.io/details/?app_id=174"><img src="https://d16t0pc4846x52.cloudfront.net/deploy.png" alt="Deploy on RepoCloud"></a> | [docs](https://repocloud.io/details/?app_id=174) |
| Alibaba Cloud| <a target="_blank" href="https://computenest.console.aliyun.com/service/instance/create/default?type=user&ServiceName=Authorizer%E7%A4%BE%E5%8C%BA%E7%89%88"><img src="https://service-info-public.oss-cn-hangzhou.aliyuncs.com/computenest-en.svg" alt="Alibaba Cloud" /></a> | [docs](https://docs.authorizer.dev/deployment/alibaba-cloud) |


### Deploy Authorizer Using Source Code

This guide helps you practice using Authorizer to evaluate it before you use it in a production environment. It includes instructions for installing the Authorizer server in local or standalone mode.

#### Prerequisites

- OS: Linux or macOS or Windows
- [Go](https://golang.org/dl/) >= 1.24 (see `go.mod`)
- [Node.js](https://nodejs.org/) >= 18 and npm (only if building the web app and dashboard)

#### Project Setup

1. Fork the [authorizer](https://github.com/authorizerdev/authorizer) repository (**Skip this step if you have access to repo**)
2. Clone repo: `git clone https://github.com/authorizerdev/authorizer.git` or use the forked url from step 1
3. Change directory: `cd authorizer`
4. Build the server binary: `make build` (or `go build -o build/authorizer .`)
5. (Optional) Build the web app and dashboard: `make build-app` and `make build-dashboard`
6. Run the server with CLI arguments:
  ```bash
   make dev
  ```
   Or run manually with all required flags:
  ```bash
  ./build/authorizer \
    --database-type=sqlite \
    --database-url=test.db \
    --jwt-type=HS256 \
    --jwt-secret=test \
    --admin-secret=admin \
    --client-id=123456 \
    --client-secret=secret
  ```
  > **v2:** The server does **not** read from `.env`. All configuration must be passed as CLI arguments. See [MIGRATION.md](MIGRATION.md) for the full mapping of env vars to flags.

### Run with Docker

The default image runs as **non-root** (UID `65532`). Writable mounts (SQLite under `/authorizer/data`, etc.) are usually **root-owned**, so pick one of:

1. **Run as root for that container** (simplest for local SQLite + volumes):

   ```sh
   docker run -p 8080:8080 -u root \
     -v authorizer_data:/authorizer/data \
     lakhansamani/authorizer \
     --database-type=sqlite \
     --database-url=/authorizer/data/data.db \
     --client-id=123456 \
     --client-secret=secret \
     --admin-secret=admin \
     --jwt-type=HS256 \
     --jwt-secret=test
   ```

2. **Keep non-root** and make the mount writable by `65532` (good for production-style bind mounts):

   ```sh
   mkdir -p ./data && sudo chown -R 65532:65532 ./data
   docker run -p 8080:8080 \
     -v "$(pwd)/data:/authorizer/data" \
     lakhansamani/authorizer \
     --database-type=sqlite \
     --database-url=/authorizer/data/data.db \
     ...
   ```

3. **Build from source** with the root target (no `-u` at run time):

   ```sh
   docker build --target final-root -t authorizer:root .
   docker run -p 8080:8080 -v authorizer_data:/authorizer/data authorizer:root \
     --database-type=sqlite --database-url=/authorizer/data/data.db ...
   ```

- Port **8080** serves the app and GraphQL; use `-p 8080:8080` to expose it.
- Volume `authorizer_data` persists the SQLite DB; use a named volume or a host path (e.g. `-v $(pwd)/data:/authorizer/data`).
- All config is passed as CLI arguments (the image uses `ENTRYPOINT ["./authorizer"]` so args after the image name go to the binary). See [MIGRATION.md](MIGRATION.md) for the full list of flags.

#### Database on your laptop (Postgres, MySQL, etc.)

Inside a container, **`localhost` / `127.0.0.1` is the container itself**, not your machine. Use a host alias instead:

- **Docker Desktop (macOS / Windows):** use `host.docker.internal` in `--database-url` or `--database-host` (built in).

  ```sh
  docker run -p 8080:8080 lakhansamani/authorizer \
    --database-type=postgres \
    --database-url="postgres://user:pass@host.docker.internal:5432/dbname?sslmode=disable" \
    ...
  ```

- **Linux (Docker Engine):** add the same hostname so it resolves to the host:

  ```sh
  docker run -p 8080:8080 --add-host=host.docker.internal:host-gateway \
    lakhansamani/authorizer \
    --database-type=postgres \
    --database-url="postgres://user:pass@host.docker.internal:5432/dbname?sslmode=disable" \
    ...
  ```

- **Alternative on Linux:** use the docker bridge gateway IP (often `172.17.0.1`) if your DB listens on `0.0.0.0`, or run with **`--network host`** so the container shares the host network (then `localhost` works; port mapping `-p` is not used the same way).

Ensure the database accepts **non-localhost** connections (e.g. `listen_addresses` in Postgres, bind address in MySQL) and that your OS firewall allows the Docker subnet.

**Extending the image with env-based config (e.g. Railway):** If you `FROM lakhansamani/authorizer` and use a shell-form `CMD` so that env vars are expanded at runtime, you must override `ENTRYPOINT` in your Dockerfile or the binary will receive `/bin/sh` and `-c` as arguments and fail. Use:

```dockerfile
FROM lakhansamani/authorizer:2.0.0-rc.1
# v2 uses CLI arguments only. Railway (etc.) inject env vars; shell form CMD expands them at runtime.
# Override ENTRYPOINT so CMD is run by a shell; otherwise the base ENTRYPOINT would receive /bin/sh -c "..." as args.
ENTRYPOINT ["/bin/sh", "-c"]
CMD ./authorizer \
  --database-type="$${DATABASE_TYPE:-postgres}" \
  --database-url="$${DATABASE_URL}" \
  --client-id="$${CLIENT_ID}" \
  --client-secret="$${CLIENT_SECRET}" \
  --admin-secret="$${ADMIN_SECRET}" \
  ...
```

Use `$$` in the Dockerfile so Docker does not expand `$VAR` at build time.

### Deploy Authorizer using binaries

Deploy / Try Authorizer using binaries. With each [Authorizer Release](https://github.com/authorizerdev/authorizer/releases), binaries are baked with required deployment files and bundled. You can download a specific version for the following operating systems:

- macOS (amd64, arm64)
- Linux (amd64, arm64)

#### Download and unzip bundle

- Download the bundle for your OS/arch from the [release page](https://github.com/authorizerdev/authorizer/releases)

> Note: For Windows, we recommend running Authorizer via Docker.

- Unzip (Mac / Linux):
  ```sh
  tar -zxf authorizer-VERSION-OS-ARCH.tar.gz
  cd authorizer-VERSION-OS-ARCH
  ```

#### Start Authorizer

- Run the binary with required CLI arguments:
  ```sh
  ./authorizer \
    --database-type=sqlite \
    --database-url=test.db \
    --jwt-type=HS256 \
    --jwt-secret=test \
    --admin-secret=admin \
    --client-id=123456 \
    --client-secret=secret
  ```

> **v2:** The binary is named `authorizer` (not `server`). Configuration is passed via CLI arguments; `.env` is not read. On macOS you may need: `xattr -d com.apple.quarantine authorizer`

## Step 2: Setup Instance

- Open the Authorizer instance endpoint in your browser
- Sign in as admin using the `--admin-secret` you configured at startup

> **v2:** Environment variables are **not** configurable from the dashboard. All configuration is set at startup via CLI arguments. See [MIGRATION.md](MIGRATION.md) for the full list of flags.

### Things to consider

- For social logins, you will need respective social platform key and secret
- For having verified users, you will need an SMTP server with an email address and password using which system can send emails. The system will send a verification link to an email address. Once an email is verified then, only able to access it.
  > Note: One can always disable the email verification to allow open sign up, which is not recommended for production as anyone can use anyone's email address 😅
- For persisting user sessions, you will need Redis URL (not in case of railway app). If you do not configure a Redis server, sessions will be persisted until the instance is up or not restarted. For better response time on authorization requests/middleware, we recommend deploying Redis on the same infra/network as your authorizer server.

## Testing

- Check the testing instructions [here](https://github.com/authorizerdev/authorizer/blob/main/.github/CONTRIBUTING.md#testing)

## Integrating into your website

This example demonstrates how you can use `[@authorizerdev/authorizer-js](/authorizer-js/getting-started)` CDN version and have login ready for your site in few seconds. You can also use the ES module version of `[@authorizerdev/authorizer-js](/authorizer-js/getting-started)` or framework-specific versions like `[@authorizerdev/authorizer-react](/authorizer-react/getting-started)`

### Copy the following code in `html` file

> **Note:** Change AUTHORIZER_URL in the below code with your authorizer URL. Also, you can change the logout button component

```html
<script src="https://unpkg.com/@authorizerdev/authorizer-js/lib/authorizer.min.js"></script>

<script type="text/javascript">
	const authorizerRef = new authorizerdev.Authorizer({
		authorizerURL: `YOUR_AUTHORIZER_INSTANCE_URL`,
		redirectURL: window.location.origin,
		clientID: 'YOUR_CLIENT_ID', // value of --client-id flag used to start the server
	});

	// use the button selector as per your application
	const logoutBtn = document.getElementById('logout');
	logoutBtn.addEventListener('click', async function () {
		await authorizerRef.logout();
		window.location.href = '/';
	});

	async function onLoad() {
		const res = await authorizerRef.authorize({
			response_type: 'code',
			use_refresh_token: false,
		});
		if (res && res.access_token) {
			// you can use user information here, eg:
			const user = await authorizerRef.getProfile({
				Authorization: `Bearer ${res.access_token}`,
			});
			const userSection = document.getElementById('user');
			const logoutSection = document.getElementById('logout-section');
			logoutSection.classList.toggle('hide');
			userSection.innerHTML = `Welcome, ${user.email}`;
		}
	}
	onLoad();
</script>
```

---

### Support my work

