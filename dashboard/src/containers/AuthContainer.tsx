import { Center, Spinner } from "@chakra-ui/react";
import React from "react";
import { Navigate, Route, Routes } from "react-router-dom";
import { useLocation } from "react-router-dom";
import { useQuery } from "urql";
import { AdminSessionQuery } from "../graphql/queries";
import { hasAdminSecret } from "../utils";

export const AuthContainer = ({ children }: { children: any }) => {
  const { pathname } = useLocation();
  const isOnboardingComplete = hasAdminSecret();
  const [result] = useQuery({
    query: AdminSessionQuery,
    pause: !isOnboardingComplete,
  });

  if (result.fetching) {
    return (
      <Center>
        <Spinner />
      </Center>
    );
  }

  if (
    result?.error?.message.includes("unauthorized") &&
    pathname !== "/login"
  ) {
    return <Navigate to="login" />;
  }

  if (!isOnboardingComplete && pathname !== "/setup") {
    return <Navigate to="setup" />;
  }

  return children;
};
