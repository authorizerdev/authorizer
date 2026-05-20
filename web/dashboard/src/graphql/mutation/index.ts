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

// Authorization mutations
export const AddResource = `
  mutation addResource($params: AddResourceInput!) {
    _add_resource(params: $params) {
      id
      name
      description
    }
  }
`;

export const UpdateResource = `
  mutation updateResource($params: UpdateResourceInput!) {
    _update_resource(params: $params) {
      id
      name
      description
    }
  }
`;

export const DeleteResource = `
  mutation deleteResource($id: ID!) {
    _delete_resource(id: $id) {
      message
    }
  }
`;

export const AddScope = `
  mutation addScope($params: AddScopeInput!) {
    _add_scope(params: $params) {
      id
      name
      description
    }
  }
`;

export const UpdateScope = `
  mutation updateScope($params: UpdateScopeInput!) {
    _update_scope(params: $params) {
      id
      name
      description
    }
  }
`;

export const DeleteScope = `
  mutation deleteScope($id: ID!) {
    _delete_scope(id: $id) {
      message
    }
  }
`;

export const AddPolicy = `
  mutation addPolicy($params: AddPolicyInput!) {
    _add_policy(params: $params) {
      id
      name
      description
      type
      logic
      decision_strategy
      targets {
        id
        target_type
        target_value
      }
    }
  }
`;

export const UpdatePolicy = `
  mutation updatePolicy($params: UpdatePolicyInput!) {
    _update_policy(params: $params) {
      id
      name
      description
      logic
      decision_strategy
      targets {
        id
        target_type
        target_value
      }
    }
  }
`;

export const DeletePolicy = `
  mutation deletePolicy($id: ID!) {
    _delete_policy(id: $id) {
      message
    }
  }
`;

export const AddPermission = `
  mutation addPermission($params: AddPermissionInput!) {
    _add_permission(params: $params) {
      id
      name
      description
      resource {
        id
        name
      }
      scopes {
        id
        name
      }
      policies {
        id
        name
      }
      decision_strategy
    }
  }
`;

export const UpdatePermission = `
  mutation updatePermission($params: UpdatePermissionInput!) {
    _update_permission(params: $params) {
      id
      name
      description
      decision_strategy
    }
  }
`;

export const DeletePermission = `
  mutation deletePermission($id: ID!) {
    _delete_permission(id: $id) {
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
