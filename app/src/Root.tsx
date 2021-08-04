import React from 'react';
import { Switch, Route } from 'react-router-dom';
import { useAuthorizer } from '@authorizerdev/authorizer-react';
import Dashboard from './pages/dashboard';
import Login from './pages/login';
import ResetPassword from './pages/rest-password';

export default function Root() {
  const { token, loading } = useAuthorizer();

  if (loading) {
    return <h1>Loading...</h1>;
  }

  if (token) {
    return (
      <Switch>
        <Route path="/app" exact>
          <Dashboard />
        </Route>
      </Switch>
    );
  }

  return (
    <Switch>
      <Route path="/app" exact>
        <Login />
      </Route>
      <Route path="/app/reset-password">
        <ResetPassword />
      </Route>
    </Switch>
  );
}
