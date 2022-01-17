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
`