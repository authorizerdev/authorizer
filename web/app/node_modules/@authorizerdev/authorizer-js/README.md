# Authorizer.js

[`@authorizerdev/authorizer-js`](https://www.npmjs.com/package/@authorizerdev/authorizer-js) is universal javaScript SDK for Authorizer API.
It supports:

- [UMD (Universal Module Definition)](https://github.com/umdjs/umd) build for browsers
- [CommonJS(cjs)](https://flaviocopes.com/commonjs/) build for NodeJS version that don't support ES Modules
- [ESM (ES Modules)](https://hacks.mozilla.org/2018/03/es-modules-a-cartoon-deep-dive/) build for modern javascript standard, i.e. ES Modules

# Migration Guide from 1.x -> 2.x

`2.x` version of `@authorizerdev/authorizer-js` has a uniform response structure that will help your applications to get right error codes and success response. Methods here have `{data, errors}` as response objects for methods of this library.

For `1.x` version of this library you can get only data in response and error would be thrown so you had to handle that in catch.

---

All the above versions require `Authorizer` instance to be instantiated and used. Instance constructor requires an object with the following keys

| Key             | Description                                                                  |
| --------------- | ---------------------------------------------------------------------------- |
| `authorizerURL` | Authorizer server endpoint                                                   |
| `redirectURL`   | URL to which you would like to redirect the user in case of successful login |

**Example**

```js
const authRef = new Authorizer({
  authorizerURL: 'https://app.herokuapp.com',
  redirectURL: window.location.origin,
});
```

## IIFE

- Step 1: Load Javascript using CDN

```html
<script src="https://unpkg.com/@authorizerdev/authorizer-js/lib/authorizer.min.js"></script>
```

- Step 2: Use the library to instantiate `Authorizer` instance and access [various methods](/authorizer-js/functions)

```html
<script type="text/javascript">
  const authorizerRef = new authorizerdev.Authorizer({
    authorizerURL: `AUTHORIZER_URL`,
    redirectURL: window.location.origin,
    clientID: 'YOUR_CLIENT_ID', // can be obtained from authorizer dashboard
  });

  // use the button selector as per your application
  const logoutBtn = document.getElementById('logout');
  logoutBtn.addEventListener('click', async function () {
    await authorizerRef.logout();
    window.location.href = '/';
  });

  async function onLoad() {
    const { data, errors } = await authorizerRef.authorize({
      response_type: 'code',
      use_refresh_token: false,
    });
    if (data && data.access_token) {
      // get user profile using the access token
      const { data: user, errors } = await authorizerRef.getProfile({
        Authorization: `Bearer ${res.access_token}`,
      });

      // 	logoutSection.classList.toggle('hide');
      // 	userSection.innerHTML = `Welcome, ${user.email}`;
    }
  }
  onLoad();
</script>
```

## CommonJS

- Step 1: Install dependencies

```sh
npm i --save @authorizerdev/authorizer-js
OR
yarn add @authorizerdev/authoirzer-js
```

- Step 2: Import and initialize the authorizer instance

```js
const { Authorizer } = require('@authorizerdev/authoirzer-js');

const authRef = new Authorizer({
  authorizerURL: 'https://app.heroku.com',
  redirectURL: 'http://app.heroku.com/app',
});

async function main() {
  await authRef.login({
    email: 'foo@bar.com',
    password: 'test',
  });
}
```

## ES Modules

- Step 1: Install dependencies

```sh
npm i --save @authorizerdev/authorizer-js
OR
yarn add @authorizerdev/authorizer-js
```

- Step 2: Import and initialize the authorizer instance

```js
import { Authorizer } from '@authorizerdev/authorizer-js';

const authRef = new Authorizer({
  authorizerURL: 'https://app.heroku.com',
  redirectURL: 'http://app.heroku.com/app',
});

async function main() {
  await authRef.login({
    email: 'foo@bar.com',
    password: 'test',
  });
}
```

## Local Development Setup

### Prerequisites

- [Pnpm](https://pnpm.io/installation)
- [NodeJS](https://nodejs.org/en/download/)

### Setup

- Clone the repository
- Install dependencies using `pnpm install`
- Run `pnpm build` to build the library
- Run `pnpm test` to run the tests

### Release

- Run `pnpm release` to release a new version of the library
