import {
  Button,
  FormControl,
  FormLabel,
  Input,
  useToast,
  VStack,
} from "@chakra-ui/react";
import React, { useEffect } from "react";
import { useMutation } from "urql";
import { AuthLayout } from "../layouts/AuthLayout";
import { AdminLogin, AdminSignup } from "../graphql/mutation";
import { useLocation, useNavigate } from "react-router-dom";

export const Auth = () => {
  const [loginResult, login] = useMutation(AdminLogin);
  const [signUpResult, signup] = useMutation(AdminSignup);

   const toast = useToast();
  const navigate = useNavigate()
  const { pathname } = useLocation();
  const isLogin = pathname === "/login";

  const handleAdminSecret = (e: any) => {
    e.preventDefault();
    const formValues = [...e.target.elements].reduce((agg: any, elem: any) => {
      if (elem.id) {
        return {
          ...agg,
          [elem.id]: elem.value,
        };
      }

      return agg;
    }, {});

    (isLogin ? login : signup)({
      secret: formValues["admin-secret"],
    }).then((res) => {
      if (!res.error?.name) {
        navigate("/");
      }
    });
  };

  const errors = isLogin ?  loginResult.error : signUpResult.error;

  useEffect(() => {
    if (errors?.graphQLErrors) {
      (errors?.graphQLErrors || []).map((error: any) => {
        toast({
          title: error.message,
          isClosable: true,
          status: "error",
          position:"bottom-right"
        });
      })
    }
  }, [errors])

  return (
    <AuthLayout>
      <form onSubmit={handleAdminSecret}>
        <VStack spacing="2.5" justify="space-between">
          <FormControl isRequired>
            <FormLabel htmlFor="admin-secret">
              {isLogin ? "Enter" : "Setup"} Admin Secret
            </FormLabel>
            <Input
              size="lg"
              id="admin-secret"
              placeholder="Admin secret"
              minLength={6}
            />
          </FormControl>
          <Button
            isLoading={signUpResult.fetching || loginResult.fetching}
            colorScheme="blue"
            size="lg"
            w="100%"
            d="block"
            type="submit"
          >
            {isLogin ? "Login" : "Sign up"}
          </Button>
        </VStack>
      </form>
    </AuthLayout>
  );
};
