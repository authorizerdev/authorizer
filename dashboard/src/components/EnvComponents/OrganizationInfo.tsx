import React from "react";
import { Flex, Stack, Center, Text, useMediaQuery } from "@chakra-ui/react";
import InputField from "../InputField";
import { TextInputType } from "../../constants";

const OrganizationInfo = ({ variables, setVariables }: any) => {
  const [isNotSmallerScreen] = useMediaQuery("(min-width:600px)");
  return (
    <div>
      {" "}
      <Text fontSize="md" paddingTop="2%" fontWeight="bold" mb={5}>
        Organization Information
      </Text>
      <Stack spacing={6} padding="2% 0%">
        <Flex direction={isNotSmallerScreen ? "row" : "column"}>
          <Flex
            w={isNotSmallerScreen ? "30%" : "40%"}
            justifyContent="start"
            alignItems="center"
          >
            <Text fontSize="sm">Organization Name:</Text>
          </Flex>
          <Center
            w={isNotSmallerScreen ? "70%" : "100%"}
            mt={isNotSmallerScreen ? "0" : "3"}
          >
            <InputField
              borderRadius={5}
              variables={variables}
              setVariables={setVariables}
              inputType={TextInputType.ORGANIZATION_NAME}
            />
          </Center>
        </Flex>
        <Flex direction={isNotSmallerScreen ? "row" : "column"}>
          <Flex
            w={isNotSmallerScreen ? "30%" : "40%"}
            justifyContent="start"
            alignItems="center"
          >
            <Text fontSize="sm">Organization Logo:</Text>
          </Flex>
          <Center
            w={isNotSmallerScreen ? "70%" : "100%"}
            mt={isNotSmallerScreen ? "0" : "3"}
          >
            <InputField
              borderRadius={5}
              variables={variables}
              setVariables={setVariables}
              inputType={TextInputType.ORGANIZATION_LOGO}
            />
          </Center>
        </Flex>
      </Stack>
    </div>
  );
};

export default OrganizationInfo;