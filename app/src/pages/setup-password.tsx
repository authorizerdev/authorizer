import React, { Fragment } from "react";
import { AuthorizerResetPassword } from "@authorizerdev/authorizer-react";

export default function SetupPassword() {
  return (
    <Fragment>
      <h1 style={{ textAlign: "center" }}>Setup new Password</h1>
      <br />
      <AuthorizerResetPassword />
    </Fragment>
  );
}
