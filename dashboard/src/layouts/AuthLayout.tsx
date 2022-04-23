import { Box, Flex, Image, Text, Spinner } from "@chakra-ui/react";
import React from "react";
import { useQuery } from "urql";
import { MetaQuery } from "../graphql/queries";

export function AuthLayout({ children }: { children: React.ReactNode }) {
  const [{ fetching, data }] = useQuery({ query: MetaQuery });
  return (
    <Flex
      flexWrap="wrap"
      h="100%"
      bg="gray.100"
      alignItems="center"
      justifyContent="center"
      flexDirection="column"
    >
      <Flex alignItems="center">
        <Image
          src="https://authorizer.dev/images/logo.png"
          alt="logo"
          height="50"
        />
        <Text fontSize="x-large" ml="3" letterSpacing="3">
          AUTHORIZER
        </Text>
      </Flex>

      {fetching ? (
        <Spinner />
      ) : (
        <>
          <Box p="6" m="5" rounded="5" bg="white" w="500px" shadow="xl">
            {children}
          </Box>
          <Text color="gray.600" fontSize="sm">
            Current Version: {data.meta.version}
          </Text>
        </>
      )}
    </Flex>
  );
}
