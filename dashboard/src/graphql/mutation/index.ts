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
  mutation updateEnvVariables($params: UpdateEnvInput!) {
    _update_env(params: $params) {
      message
    }
  }
`;

export const UpdateUser = `
  mutation updateUser($params: UpdateUserInput!) {
    _update_user(params: $params) {
      id
    }
  }
`;

export const DeleteUser = `
  mutation deleteUser($params: DeleteUserInput!) {
    _delete_user(params: $params) {
      message
    }
  }
`;

export const InviteMembers = `
  mutation inviteMembers($params: InviteMemberInput!) {
    _invite_members(params: $params) {
      message
    }
  }
`;

export const RevokeAccess = `
  mutation revokeAccess($param: UpdateAccessInput!) {
    _revoke_access(param: $param) {
      message
    }
  }
`;

export const EnableAccess = `
  mutation revokeAccess($param: UpdateAccessInput!) {
    _enable_access(param: $param) {
      message
    }
  }
`;

export const GenerateKeys = `
  mutation generateKeys($params: GenerateJWTKeysInput!) {
    _generate_jwt_keys(params: $params) {
      secret
      public_key
      private_key
    }
  }
`;

export const AddWebhook = `
  mutation addWebhook($params: AddWebhookRequest!) {
    _add_webhook(params: $params) {
      message
    }
  }
`;

export const EditWebhook = `
  mutation editWebhook($params: UpdateWebhookRequest!) {
    _update_webhook(params: $params) {
      message
    }
  }
`;
