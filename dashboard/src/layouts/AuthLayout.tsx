import { Box, Center, Flex, Image, Text } from "@chakra-ui/react";
import React from "react";
import { LOGO_URL } from "../constants";

export function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <Flex flexWrap="wrap" h="100%">
      <Center h="100%" flex="3" bg="blue.500" flexDirection="column">
        <Image
          src={LOGO_URL}
          alt=""
        />

        <Text
          color="white"
          casing="uppercase"
          fontSize="3xl"
          mt="2"
          letterSpacing="2.25px"
        >
          Authorizer
        </Text>
      </Center>
      <Center h="100%" flex="2">
        {children}
      </Center>
    </Flex>
  );
}
