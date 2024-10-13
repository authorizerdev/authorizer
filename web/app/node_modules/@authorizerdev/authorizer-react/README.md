# authorizer-react

Authorizer React SDK allows you to implement authentication in your [React](https://reactjs.org/) application quickly. It also allows you to access the user profile.

Here is a quick guide on getting started with `@authorizerdev/authorizer-react` package.

## Code Sandbox Demo: https://codesandbox.io/s/authorizer-demo-qgjpw

## Step 1 - Create Instance

Get Authorizer URL by instantiating [Authorizer instance](/deployment) and configuring it with necessary [environment variables](/core/env).

## Step 2 - Install package

Install `@authorizerdev/authorizer-react` library

```sh
npm i --save @authorizerdev/authorizer-react
OR
yarn add @authorizerdev/authorizer-react
```

## Step 3 - Configure Provider and use Authorizer Components

Authorizer comes with [react context](https://reactjs.org/docs/context.html) which serves as `Provider` component for the application

```jsx
import {
  AuthorizerProvider,
  Authorizer,
  useAuthorizer,
} from '@authorizerdev/authorizer-react';

const App = () => {
  return (
    <AuthorizerProvider
      config={{
        authorizerURL: 'http://localhost:8080',
        redirectURL: window.location.origin,
        clientID: 'YOUR_CLIENT_ID',
      }}
    >
      <LoginSignup />
      <Profile />
    </AuthorizerProvider>
  );
};

const LoginSignup = () => {
  return <Authorizer />;
};

const Profile = () => {
  const { user } = useAuthorizer();

  if (user) {
    return <div>{user.email}</div>;
  }

  return null;
};
```

## Commands

### Local Development

The recommended workflow is to run authorizer in one terminal:

```bash
npm start # or yarn start
```

This builds to `/dist` and runs the project in watch mode so any edits you save inside `src` causes a rebuild to `/dist`.

Then run either Storybook or the example playground:

### Example

Then run the example inside another:

```bash
cd example
npm i # or yarn to install dependencies
npm start # or yarn start
```

The default example imports and live reloads whatever is in `/dist`, so if you are seeing an out of date component, make sure TSDX is running in watch mode like we recommend above. **No symlinking required**, we use [Parcel's aliasing](https://parceljs.org/module_resolution.html#aliases).

To do a one-off build, use `npm run build` or `yarn build`.

To run tests, use `npm test` or `yarn test`.

## Configuration

Code quality is set up for you with `prettier`, `husky`, and `lint-staged`. Adjust the respective fields in `package.json` accordingly.

### Jest

Jest tests are set up to run with `npm test` or `yarn test`.

### Bundle analysis

Calculates the real cost of your library using [size-limit](https://github.com/ai/size-limit) with `npm run size` and visulize it with `npm run analyze`.
