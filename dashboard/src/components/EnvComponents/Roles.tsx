import React from "react";
import { Flex, Stack, Center, Text, useMediaQuery } from "@chakra-ui/react";
import { ArrayInputType } from "../../constants";
import InputField from "../InputField";

const Roles = ({ variables, setVariables }: any) => {
  const [isNotSmallerScreen] = useMediaQuery("(min-width:600px)");
  return (
    <div>
      {" "}
      <Text fontSize="md" paddingTop="2%" fontWeight="bold" mb={5}>
        Roles
      </Text>
      <Stack spacing={6} padding="2% 0%">
        <Flex direction={isNotSmallerScreen ? "row" : "column"}>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Roles:</Text>
          </Flex>
          <Center
            w={isNotSmallerScreen ? "70%" : "100%"}
            mt={isNotSmallerScreen ? "0" : "2"}
            overflow="hidden"
          >
            <InputField
              borderRadius={7}
              variables={variables}
              setVariables={setVariables}
              inputType={ArrayInputType.ROLES}
            />
          </Center>
        </Flex>
        <Flex direction={isNotSmallerScreen ? "row" : "column"}>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Default Roles:</Text>
          </Flex>
          <Center
            w={isNotSmallerScreen ? "70%" : "100%"}
            mt={isNotSmallerScreen ? "0" : "2"}
          >
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={ArrayInputType.DEFAULT_ROLES}
            />
          </Center>
        </Flex>
        <Flex direction={isNotSmallerScreen ? "row" : "column"}>
          <Flex w="30%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Protected Roles:</Text>
          </Flex>
          <Center
            w={isNotSmallerScreen ? "70%" : "100%"}
            mt={isNotSmallerScreen ? "0" : "2"}
          >
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={ArrayInputType.PROTECTED_ROLES}
            />
          </Center>
        </Flex>
      </Stack>
    </div>
  );
};

export default Roles;