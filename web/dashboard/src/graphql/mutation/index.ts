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

export const FgaWriteModel = `
  mutation fgaWriteModel($params: FgaWriteModelInput!) {
    _fga_write_model(params: $params) {
      id
      dsl
    }
  }
`;

export const FgaWriteTuples = `
  mutation fgaWriteTuples($params: FgaWriteTuplesInput!) {
    _fga_write_tuples(params: $params) {
      message
    }
  }
`;

export const FgaDeleteTuples = `
  mutation fgaDeleteTuples($params: FgaWriteTuplesInput!) {
    _fga_delete_tuples(params: $params) {
      message
    }
  }
`;

export const FgaReset = `
  mutation fgaReset {
    _fga_reset {
      message
    }
  }
`;

export const CreateClient = `
  mutation createClient($params: CreateClientRequest!) {
    _create_client(params: $params) {
      client {
        id
        name
      }
      client_secret
    }
  }
`;

export const UpdateClient = `
  mutation updateClient($params: UpdateClientRequest!) {
    _update_client(params: $params) {
      id
      name
    }
  }
`;

export const DeleteClient = `
  mutation deleteClient($params: ClientRequest!) {
    _delete_client(params: $params) {
      message
    }
  }
`;

export const RotateClientSecret = `
  mutation rotateClientSecret($params: ClientRequest!) {
    _rotate_client_secret(params: $params) {
      client {
        id
        name
      }
      client_secret
    }
  }
`;

export const AddTrustedIssuer = `
  mutation addTrustedIssuer($params: AddTrustedIssuerRequest!) {
    _add_trusted_issuer(params: $params) {
      id
      name
    }
  }
`;

export const UpdateTrustedIssuer = `
  mutation updateTrustedIssuer($params: UpdateTrustedIssuerRequest!) {
    _update_trusted_issuer(params: $params) {
      id
      name
    }
  }
`;

export const DeleteTrustedIssuer = `
  mutation deleteTrustedIssuer($params: TrustedIssuerRequest!) {
    _delete_trusted_issuer(params: $params) {
      message
    }
  }
`;

export const CreateOrganization = `
  mutation createOrganization($params: CreateOrganizationRequest!) {
    _create_organization(params: $params) {
      id
      name
    }
  }
`;

export const UpdateOrganization = `
  mutation updateOrganization($params: UpdateOrganizationRequest!) {
    _update_organization(params: $params) {
      id
      name
    }
  }
`;

export const DeleteOrganization = `
  mutation deleteOrganization($params: OrganizationRequest!) {
    _delete_organization(params: $params) {
      message
    }
  }
`;

export const AddOrgMember = `
  mutation addOrgMember($params: AddOrgMemberRequest!) {
    _add_org_member(params: $params) {
      id
      user_id
    }
  }
`;

export const RemoveOrgMember = `
  mutation removeOrgMember($params: RemoveOrgMemberRequest!) {
    _remove_org_member(params: $params) {
      message
    }
  }
`;

export const CreateOrgOIDCConnection = `
  mutation createOrgOIDCConnection($params: CreateOrgOIDCConnectionRequest!) {
    _create_org_oidc_connection(params: $params) {
      id
      name
    }
  }
`;

export const UpdateOrgOIDCConnection = `
  mutation updateOrgOIDCConnection($params: UpdateOrgOIDCConnectionRequest!) {
    _update_org_oidc_connection(params: $params) {
      id
      name
    }
  }
`;

export const DeleteOrgOIDCConnection = `
  mutation deleteOrgOIDCConnection($params: OrgOIDCConnectionRequest!) {
    _delete_org_oidc_connection(params: $params) {
      message
    }
  }
`;

export const CreateOrgSAMLConnection = `
  mutation createOrgSAMLConnection($params: CreateOrgSAMLConnectionRequest!) {
    _create_org_saml_connection(params: $params) {
      id
      name
    }
  }
`;

export const UpdateOrgSAMLConnection = `
  mutation updateOrgSAMLConnection($params: UpdateOrgSAMLConnectionRequest!) {
    _update_org_saml_connection(params: $params) {
      id
      name
    }
  }
`;

export const DeleteOrgSAMLConnection = `
  mutation deleteOrgSAMLConnection($params: OrgSAMLConnectionRequest!) {
    _delete_org_saml_connection(params: $params) {
      message
    }
  }
`;

export const CreateScimEndpoint = `
  mutation createScimEndpoint($params: CreateScimEndpointRequest!) {
    _create_scim_endpoint(params: $params) {
      scim_endpoint {
        id
        org_id
        enabled
      }
      token
    }
  }
`;

export const RotateScimToken = `
  mutation rotateScimToken($params: ScimEndpointRequest!) {
    _rotate_scim_token(params: $params) {
      scim_endpoint {
        id
        org_id
        enabled
      }
      token
    }
  }
`;

export const DeleteScimEndpoint = `
  mutation deleteScimEndpoint($params: ScimEndpointRequest!) {
    _delete_scim_endpoint(params: $params) {
      message
    }
  }
`;
