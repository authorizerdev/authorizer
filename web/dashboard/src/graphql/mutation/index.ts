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

export const UpdateUser = `
  mutation updateUser($params: UpdateUserRequest!) {
    _update_user(params: $params) {
      id
    }
  }
`;

export const DeleteUser = `
  mutation deleteUser($params: DeleteUserRequest!) {
    _delete_user(params: $params) {
      message
    }
  }
`;

export const InviteMembers = `
  mutation inviteMembers($params: InviteMemberRequest!) {
    _invite_members(params: $params) {
      message
    }
  }
`;

export const RevokeAccess = `
  mutation revokeAccess($param: UpdateAccessRequest!) {
    _revoke_access(param: $param) {
      message
    }
  }
`;

export const EnableAccess = `
  mutation revokeAccess($param: UpdateAccessRequest!) {
    _enable_access(param: $param) {
      message
    }
  }
`;

export const GenerateKeys = `
  mutation generateKeys($params: GenerateJWTKeysRequest!) {
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

export const DeleteWebhook = `
  mutation deleteWebhook($params: WebhookRequest!) {
    _delete_webhook(params: $params) {
      message
    }
  }
`;

export const TestEndpoint = `
  mutation testEndpoint($params: TestEndpointRequest!) {
    _test_endpoint(params: $params) {
      http_status
      response
    }
  }
`;

export const AddEmailTemplate = `
  mutation addEmailTemplate($params: AddEmailTemplateRequest!) {
    _add_email_template(params: $params) {
      message
    }
  }
`;

export const EditEmailTemplate = `
  mutation editEmailTemplate($params: UpdateEmailTemplateRequest!) {
    _update_email_template(params: $params) {
      message
    }
  }
`;

export const DeleteEmailTemplate = `
  mutation deleteEmailTemplate($params: DeleteEmailTemplateRequest!) {
    _delete_email_template(params: $params) {
      message
    }
  }
`;
