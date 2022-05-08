import React from "react";
import { Flex, Stack, Text } from "@chakra-ui/react";
import InputField from "../InputField";
import { SwitchInputType } from "../../constants";

const UICustomization = ({ variables, setVariables }: any) => {
  return (
    <div>
      {" "}
      <Text fontSize="md" paddingTop="2%" fontWeight="bold" mb={5}>
        Disable Features
      </Text>
      <Stack spacing={6} padding="2% 0%">
        <Flex>
          <Flex w="100%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Disable Login Page:</Text>
          </Flex>
          <Flex justifyContent="start">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={SwitchInputType.DISABLE_LOGIN_PAGE}
            />
          </Flex>
        </Flex>
        <Flex>
          <Flex w="100%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Disable Email Verification:</Text>
          </Flex>
          <Flex justifyContent="start">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={SwitchInputType.DISABLE_EMAIL_VERIFICATION}
            />
          </Flex>
        </Flex>
        <Flex>
          <Flex w="100%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Disable Magic Login Link:</Text>
          </Flex>
          <Flex justifyContent="start">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={SwitchInputType.DISABLE_MAGIC_LINK_LOGIN}
            />
          </Flex>
        </Flex>
        <Flex>
          <Flex w="100%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Disable Basic Authentication:</Text>
          </Flex>
          <Flex justifyContent="start">
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={SwitchInputType.DISABLE_BASIC_AUTHENTICATION}
            />
          </Flex>
        </Flex>
        <Flex>
          <Flex w="100%" justifyContent="start" alignItems="center">
            <Text fontSize="sm">Disable Sign Up:</Text>
          </Flex>
          <Flex justifyContent="start" mb={3}>
            <InputField
              variables={variables}
              setVariables={setVariables}
              inputType={SwitchInputType.DISABLE_SIGN_UP}
            />
          </Flex>
        </Flex>
      </Stack>
    </div>
  );
};

export default UICustomization;