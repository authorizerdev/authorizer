export const AdminSignup = `
 mutation adminSignup($secret: String!) {
    _admin_signup (params: {admin_secret: $secret}) {
      message
    }
  }
`;

export const AdminLogin = `
mutation adminLogin($secret: String!){
  _admin_login(params: { admin_secret: $secret }) {
    message
  }
}
`;

export const AdminLogout = `
  mutation adminLogout {
    _admin_logout {
      message
    }
  }
`;

export const UpdateEnvVariables = `
  mutation updateEnvVariables(
    $params: UpdateEnvInput!
    ) {
    _update_env(params: $params) {
      message
    }
  }
`;

export const UpdateUser = `
  mutation updateUser(
    $params: UpdateUserInput!
    ) {
    _update_user(params: $params) {
      id
    }
  }
`;
