import React from "react";
import { useAuthorizer } from "@authorizerdev/authorizer-react";

export default function Dashboard() {
  const [loading, setLoading] = React.useState(false);
  const { user, setToken, authorizerRef } = useAuthorizer();

  const onLogout = async () => {
    setLoading(true);
    await authorizerRef.logout();
    setToken(null);
    setLoading(false);
  };

  return (
    <div>
      <h1>Hey 👋,</h1>
      <p>Thank you for using authorizer.</p>
      <p>
        Your email address is{" "}
        <a href={`mailto:${user?.email}`} style={{ color: "#3B82F6" }}>
          {user?.email}
        </a>
      </p>

      <br />
      {loading ? (
        <h3>Processing....</h3>
      ) : (
        <h3
          style={{
            color: "#3B82F6",
            cursor: "pointer",
          }}
          onClick={onLogout}
        >
          Logout
        </h3>
      )}
    </div>
  );
}
